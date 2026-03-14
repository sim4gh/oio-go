package cli

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/atotto/clipboard"
	"github.com/briandowns/spinner"
	"github.com/mdp/qrterminal/v3"
	"github.com/sim4gh/oio-go/internal/whatsapp"
	"github.com/spf13/cobra"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/proto/waWeb"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

var waLsAll bool

func addWaCommands() {
	waCmd := &cobra.Command{
		Use:   "wa",
		Short: "WhatsApp messaging commands",
		Long: `WhatsApp messaging commands

Examples:
  oio wa link                          Link WhatsApp (scan QR code)
  oio wa send 7778887788 "Hello!"      Send a message
  oio wa send 7778887788               Send clipboard content
  oio wa ls                            Show unread messages
  oio wa ls --all                      Show all recent conversations
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
		Short:   "Show unread messages",
		Aliases: []string{"list"},
		Long: `Connect to WhatsApp and display unread messages from history sync.

By default, only shows conversations with unread messages (the red badge).
Use --all to show all recent conversations.

Examples:
  oio wa ls          # show unread messages
  oio wa ls --all    # show all recent conversations`,
		RunE: runWaLs,
	}
	lsCmd.Flags().BoolVar(&waLsAll, "all", false, "Show all recent conversations, not just unread")

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

func runWaLink(cmd *cobra.Command, args []string) error {
	client, err := whatsapp.NewClient(false)
	if err != nil {
		return fmt.Errorf("failed to initialize WhatsApp: %w", err)
	}
	defer client.Disconnect()

	if client.Store.ID != nil {
		fmt.Println("WhatsApp is already linked.")
		fmt.Println("Run \"oio wa unlink\" first to re-link.")
		return nil
	}

	// Listen for pairing events via event handler (primary success signal)
	pairSuccess := make(chan string, 1)
	pairError := make(chan error, 1)

	client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.PairSuccess:
			select {
			case pairSuccess <- v.ID.String():
			default:
			}
		case *events.PairError:
			select {
			case pairError <- fmt.Errorf("pairing failed: %v", v.Error):
			default:
			}
		case *events.Connected:
			select {
			case pairSuccess <- "connected":
			default:
			}
		}
	})

	qrChan, err := client.GetQRChannel(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get QR channel: %w", err)
	}

	err = client.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	fmt.Println("\nScan this QR code with WhatsApp:")
	fmt.Println("  WhatsApp > Settings > Linked Devices > Link a Device")
	fmt.Println()

	for {
		select {
		case evt, ok := <-qrChan:
			if !ok {
				// QR channel closed — check if we paired via event handler
				select {
				case <-pairSuccess:
					fmt.Println("\nWhatsApp linked successfully!")
					return nil
				default:
					return fmt.Errorf("connection closed unexpectedly")
				}
			}
			switch evt.Event {
			case "code":
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			case "login":
				fmt.Println("\nWhatsApp linked successfully!")
				return nil
			case "timeout":
				return fmt.Errorf("QR code expired. Run \"oio wa link\" again")
			case "error":
				return fmt.Errorf("QR channel error. Run \"oio wa link\" again")
			}

		case <-pairSuccess:
			fmt.Println("\nWhatsApp linked successfully!")
			// Wait briefly for the session to be fully saved
			time.Sleep(1 * time.Second)
			return nil

		case pairErr := <-pairError:
			return pairErr
		}
	}
}

func runWaSend(cmd *cobra.Command, args []string) error {
	client, err := whatsapp.NewClient(false)
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

// unreadConversation holds a conversation with unread messages from history sync.
type unreadConversation struct {
	Name        string
	JID         string
	UnreadCount uint32
	Messages    []*waWeb.WebMessageInfo
	LastMsgTime uint64
}

func runWaLs(cmd *cobra.Command, args []string) error {
	client, err := whatsapp.NewClient(false)
	if err != nil {
		return err
	}

	if client.Store.ID == nil {
		return fmt.Errorf("WhatsApp not linked. Run \"oio wa link\" first")
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Connecting and syncing messages..."
	s.Start()

	var mu sync.Mutex
	var conversations []*unreadConversation

	client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.HistorySync:
			if v.Data == nil {
				return
			}
			mu.Lock()
			defer mu.Unlock()
			for _, conv := range v.Data.GetConversations() {
				if !waLsAll && conv.GetUnreadCount() == 0 {
					continue
				}
				conversations = append(conversations, &unreadConversation{
					Name:        conversationName(conv),
					JID:         conv.GetID(),
					UnreadCount: conv.GetUnreadCount(),
					Messages:    extractWebMessages(conv.GetMessages()),
					LastMsgTime: conv.GetLastMsgTimestamp(),
				})
			}
		}
	})

	err = client.Connect()
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Disconnect()

	// Wait for history sync events (they arrive in chunks over several seconds)
	time.Sleep(10 * time.Second)
	s.Stop()

	mu.Lock()
	defer mu.Unlock()

	if len(conversations) == 0 {
		if waLsAll {
			fmt.Println("No conversations found.")
		} else {
			fmt.Println("No unread messages.")
		}
		return nil
	}

	// Sort by last message timestamp (most recent first)
	sort.Slice(conversations, func(i, j int) bool {
		return conversations[i].LastMsgTime > conversations[j].LastMsgTime
	})

	for _, conv := range conversations {
		printConversation(conv)
	}

	return nil
}

// conversationName returns the best display name for a conversation.
func conversationName(conv *waHistorySync.Conversation) string {
	if n := conv.GetDisplayName(); n != "" {
		return n
	}
	if n := conv.GetName(); n != "" {
		return n
	}
	return conv.GetID()
}

// extractWebMessages extracts WebMessageInfo from history sync messages.
func extractWebMessages(msgs []*waHistorySync.HistorySyncMsg) []*waWeb.WebMessageInfo {
	out := make([]*waWeb.WebMessageInfo, 0, len(msgs))
	for _, m := range msgs {
		if wmi := m.GetMessage(); wmi != nil {
			out = append(out, wmi)
		}
	}
	return out
}

// printConversation renders a single conversation's unread messages.
func printConversation(conv *unreadConversation) {
	isGroup := strings.Contains(conv.JID, "@g.us")

	// Header line
	if conv.UnreadCount > 0 {
		fmt.Printf("\n%s — %d unread\n", conv.Name, conv.UnreadCount)
	} else {
		fmt.Printf("\n%s\n", conv.Name)
	}

	// Sort messages by timestamp (oldest first)
	msgs := conv.Messages
	sort.Slice(msgs, func(i, j int) bool {
		return msgs[i].GetMessageTimestamp() < msgs[j].GetMessageTimestamp()
	})

	// Show up to unreadCount messages from the end, or all if --all
	showMsgs := msgs
	if !waLsAll && conv.UnreadCount > 0 && int(conv.UnreadCount) < len(msgs) {
		showMsgs = msgs[len(msgs)-int(conv.UnreadCount):]
	}

	for _, wmi := range showMsgs {
		if wmi.GetKey().GetFromMe() {
			continue // Skip our own sent messages
		}
		text := extractMessageText(wmi.GetMessage())
		if text == "" {
			continue
		}

		ts := time.Unix(int64(wmi.GetMessageTimestamp()), 0).Local().Format("15:04")

		if isGroup {
			sender := wmi.GetPushName()
			if sender == "" {
				sender = wmi.GetParticipant()
			}
			if sender != "" {
				fmt.Printf("  [%s] %s: %s\n", ts, sender, truncateMsg(text, 70))
			} else {
				fmt.Printf("  [%s] %s\n", ts, truncateMsg(text, 70))
			}
		} else {
			fmt.Printf("  [%s] %s\n", ts, truncateMsg(text, 70))
		}
	}
}

func runWaUnlink(cmd *cobra.Command, args []string) error {
	if !whatsapp.IsLinked() {
		fmt.Println("WhatsApp is not linked.")
		return nil
	}

	client, err := whatsapp.NewClient(false)
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

	client, err := whatsapp.NewClient(false)
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

// extractMessageText extracts user-visible text from a WhatsApp message.
// Returns empty string for protocol/system messages that should be skipped.
func extractMessageText(msg *waE2E.Message) string {
	if msg == nil {
		return ""
	}

	// Unwrap container message types
	if msg.GetEphemeralMessage() != nil && msg.GetEphemeralMessage().GetMessage() != nil {
		msg = msg.GetEphemeralMessage().GetMessage()
	}
	if msg.GetDeviceSentMessage() != nil && msg.GetDeviceSentMessage().GetMessage() != nil {
		msg = msg.GetDeviceSentMessage().GetMessage()
	}
	if msg.GetViewOnceMessage() != nil && msg.GetViewOnceMessage().GetMessage() != nil {
		msg = msg.GetViewOnceMessage().GetMessage()
	}
	if msg.GetViewOnceMessageV2() != nil && msg.GetViewOnceMessageV2().GetMessage() != nil {
		msg = msg.GetViewOnceMessageV2().GetMessage()
	}

	// Extract text from known message types
	if t := msg.GetConversation(); t != "" {
		return t
	}
	if t := msg.GetExtendedTextMessage(); t != nil && t.GetText() != "" {
		return t.GetText()
	}
	if t := msg.GetImageMessage(); t != nil {
		if cap := t.GetCaption(); cap != "" {
			return "[Image] " + cap
		}
		return "[Image]"
	}
	if t := msg.GetVideoMessage(); t != nil {
		if cap := t.GetCaption(); cap != "" {
			return "[Video] " + cap
		}
		return "[Video]"
	}
	if msg.GetDocumentMessage() != nil {
		name := msg.GetDocumentMessage().GetFileName()
		if name != "" {
			return "[Document] " + name
		}
		return "[Document]"
	}
	if msg.GetAudioMessage() != nil {
		if msg.GetAudioMessage().GetPTT() {
			return "[Voice note]"
		}
		return "[Audio]"
	}
	if msg.GetStickerMessage() != nil {
		return "[Sticker]"
	}
	if msg.GetContactMessage() != nil {
		return "[Contact] " + msg.GetContactMessage().GetDisplayName()
	}
	if msg.GetLocationMessage() != nil {
		return "[Location]"
	}
	if msg.GetLiveLocationMessage() != nil {
		return "[Live location]"
	}
	if msg.GetReactionMessage() != nil {
		return msg.GetReactionMessage().GetText()
	}
	if msg.GetPollCreationMessage() != nil {
		return "[Poll] " + msg.GetPollCreationMessage().GetName()
	}

	// Skip protocol/system messages (no user content)
	if msg.GetProtocolMessage() != nil ||
		msg.GetSenderKeyDistributionMessage() != nil ||
		msg.GetPollUpdateMessage() != nil {
		return ""
	}

	// Unknown message type — skip rather than show "[Message]"
	return ""
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
