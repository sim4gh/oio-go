package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/briandowns/spinner"
	"github.com/mdp/qrterminal/v3"
	"github.com/olekukonko/tablewriter"
	"github.com/sim4gh/oio-go/internal/whatsapp"
	"github.com/spf13/cobra"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func addWaCommands() {
	waCmd := &cobra.Command{
		Use:   "wa",
		Short: "WhatsApp messaging commands",
		Long: `WhatsApp messaging commands

Examples:
  oio wa link                          Link WhatsApp (scan QR code)
  oio wa send 7778887788 "Hello!"      Send a message
  oio wa send 7778887788               Send clipboard content
  oio wa ls                            List recent messages
  oio wa status                        Check link status
  oio wa unlink                        Unlink WhatsApp`,
	}

	linkCmd := &cobra.Command{
		Use:   "link",
		Short: "Link WhatsApp by scanning QR code",
		RunE:  runWaLink,
	}

	sendCmd := &cobra.Command{
		Use:   "send <number> [message]",
		Short: "Send a WhatsApp message",
		Long: `Send a WhatsApp message to a phone number.

If no message is provided, sends clipboard content.
Phone number should include country code (e.g., 14255687870 for US).

Examples:
  oio wa send 14255687870 "Hello!"
  oio wa send 14255687870              # sends clipboard
  oio wa send +1-425-568-7870 "Hi"     # non-digits are stripped`,
		Args: cobra.MinimumNArgs(1),
		RunE: runWaSend,
	}

	lsCmd := &cobra.Command{
		Use:     "ls",
		Short:   "Show incoming messages (live)",
		Aliases: []string{"list"},
		Long: `Connect to WhatsApp and display incoming messages in real-time.

Listens for messages for the specified duration (default: 10 seconds).
Press Ctrl+C to stop early.

Examples:
  oio wa ls               # listen for 10 seconds
  oio wa ls --duration 30 # listen for 30 seconds`,
		RunE: runWaLs,
	}
	lsCmd.Flags().IntVar(&waLsDuration, "duration", 10, "Seconds to listen for messages")

	unlinkCmd := &cobra.Command{
		Use:   "unlink",
		Short: "Unlink WhatsApp and clear session",
		RunE:  runWaUnlink,
	}

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Check WhatsApp link status",
		RunE:  runWaStatus,
	}

	waCmd.AddCommand(linkCmd, sendCmd, lsCmd, unlinkCmd, statusCmd)
	rootCmd.AddCommand(waCmd)
}

var waLsDuration int

func runWaLink(cmd *cobra.Command, args []string) error {
	client, err := whatsapp.NewClient()
	if err != nil {
		return fmt.Errorf("failed to initialize WhatsApp: %w", err)
	}
	defer client.Disconnect()

	if client.Store.ID != nil {
		fmt.Println("WhatsApp is already linked.")
		fmt.Println("Run \"oio wa unlink\" first to re-link.")
		return nil
	}

	qrChan, err := client.GetQRChannel(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get QR channel: %w", err)
	}

	err = client.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	fmt.Println("\nScan this QR code with WhatsApp:")
	fmt.Println("  WhatsApp > Settings > Linked Devices > Link a Device\n")

	for evt := range qrChan {
		switch evt.Event {
		case "code":
			qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
		case "login":
			fmt.Println("\nWhatsApp linked successfully!")
			return nil
		case "timeout":
			return fmt.Errorf("QR code expired. Run \"oio wa link\" again")
		}
	}

	return nil
}

func runWaSend(cmd *cobra.Command, args []string) error {
	client, err := whatsapp.NewClient()
	if err != nil {
		return err
	}

	if client.Store.ID == nil {
		return fmt.Errorf("WhatsApp not linked. Run \"oio wa link\" first")
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Connecting to WhatsApp..."
	s.Start()

	err = client.Connect()
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Disconnect()

	s.Stop()

	number := args[0]
	var message string
	if len(args) > 1 {
		message = strings.Join(args[1:], " ")
	} else {
		text, clipErr := clipboard.ReadAll()
		if clipErr != nil || strings.TrimSpace(text) == "" {
			return fmt.Errorf("no message provided and clipboard is empty")
		}
		message = text
		fmt.Printf("Sending clipboard content (%d chars)\n", len(message))
	}

	jid := whatsapp.FormatNumber(number)

	s = spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Sending message..."
	s.Start()

	_, err = client.SendMessage(context.Background(), jid, &waE2E.Message{
		Conversation: proto.String(message),
	})
	s.Stop()

	if err != nil {
		return fmt.Errorf("failed to send: %w", err)
	}

	fmt.Printf("Message sent to %s\n", number)
	return nil
}

func runWaLs(cmd *cobra.Command, args []string) error {
	client, err := whatsapp.NewClient()
	if err != nil {
		return err
	}

	if client.Store.ID == nil {
		return fmt.Errorf("WhatsApp not linked. Run \"oio wa link\" first")
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Connecting to WhatsApp..."
	s.Start()

	type chatMessage struct {
		Time   string
		From   string
		Text   string
		IsGroup bool
	}
	var messages []chatMessage

	client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			sender := v.Info.Sender.User
			if v.Info.PushName != "" {
				sender = v.Info.PushName
			}

			text := ""
			if v.Message.GetConversation() != "" {
				text = v.Message.GetConversation()
			} else if v.Message.GetExtendedTextMessage() != nil {
				text = v.Message.GetExtendedTextMessage().GetText()
			} else if v.Message.GetImageMessage() != nil {
				text = "[Image]"
				if cap := v.Message.GetImageMessage().GetCaption(); cap != "" {
					text = "[Image] " + cap
				}
			} else if v.Message.GetVideoMessage() != nil {
				text = "[Video]"
			} else if v.Message.GetDocumentMessage() != nil {
				text = "[Document]"
			} else if v.Message.GetAudioMessage() != nil {
				text = "[Audio]"
			} else if v.Message.GetStickerMessage() != nil {
				text = "[Sticker]"
			} else {
				text = "[Message]"
			}

			isGroup := v.Info.IsGroup
			messages = append(messages, chatMessage{
				Time:    v.Info.Timestamp.Local().Format("15:04:05"),
				From:    sender,
				Text:    text,
				IsGroup: isGroup,
			})

			// Print in real-time
			s.Stop()
			groupTag := ""
			if isGroup {
				groupTag = " [group]"
			}
			fmt.Printf("[%s] %s%s: %s\n", v.Info.Timestamp.Local().Format("15:04"), sender, groupTag, truncateMsg(text, 80))
		}
	})

	err = client.Connect()
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Disconnect()

	s.Stop()

	duration := time.Duration(waLsDuration) * time.Second
	fmt.Printf("Listening for messages (%ds)... Press Ctrl+C to stop.\n\n", waLsDuration)

	time.Sleep(duration)

	if len(messages) == 0 {
		fmt.Println("\nNo messages received during this period.")
	} else {
		fmt.Printf("\n%d message(s) received.\n", len(messages))
	}

	return nil
}

func runWaUnlink(cmd *cobra.Command, args []string) error {
	if !whatsapp.IsLinked() {
		fmt.Println("WhatsApp is not linked.")
		return nil
	}

	client, err := whatsapp.NewClient()
	if err != nil {
		// If we can't create client, just delete the DB
		if delErr := whatsapp.DeleteDB(); delErr != nil {
			return fmt.Errorf("failed to delete session: %w", delErr)
		}
		fmt.Println("WhatsApp session cleared.")
		return nil
	}

	// Try to properly logout from WhatsApp servers
	if client.Store.ID != nil {
		if connErr := client.Connect(); connErr == nil {
			_ = client.Logout(context.Background())
		}
		client.Disconnect()
	}

	if err := whatsapp.DeleteDB(); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	fmt.Println("WhatsApp unlinked successfully.")
	return nil
}

func runWaStatus(cmd *cobra.Command, args []string) error {
	if !whatsapp.IsLinked() {
		fmt.Println("WhatsApp: Not linked")
		fmt.Println("Run \"oio wa link\" to connect your WhatsApp account.")
		return nil
	}

	client, err := whatsapp.NewClient()
	if err != nil {
		return err
	}

	if client.Store.ID == nil {
		fmt.Println("WhatsApp: Not linked (empty session)")
		fmt.Println("Run \"oio wa link\" to connect your WhatsApp account.")
		return nil
	}

	fmt.Println("WhatsApp: Linked")
	fmt.Printf("  Device: %s\n", client.Store.ID.String())

	// Try a quick connect to verify session is still valid
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Verifying connection..."
	s.Start()

	err = client.Connect()
	s.Stop()

	if err != nil {
		fmt.Println("  Status: Session expired (re-link with \"oio wa link\")")
	} else {
		fmt.Println("  Status: Connected")
		client.Disconnect()
	}

	return nil
}

// displayContacts shows contacts in a table
func displayContacts(contacts map[string]string) {
	if len(contacts) == 0 {
		fmt.Println("No contacts found.")
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Number"})
	table.SetBorder(true)
	table.SetAutoWrapText(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for number, name := range contacts {
		if name == "" {
			name = number
		}
		table.Append([]string{name, number})
	}

	table.Render()
}

func truncateMsg(s string, maxLen int) string {
	// Replace newlines with spaces for single-line display
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}
