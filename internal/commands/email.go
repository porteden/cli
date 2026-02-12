package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/porteden/cli/internal/api"
	"github.com/porteden/cli/internal/output"
	"github.com/spf13/cobra"
)

var emailCmd = &cobra.Command{
	Use:     "email",
	Short:   "Email commands",
	Aliases: []string{"mail"},
}

var messagesCmd = &cobra.Command{
	Use:   "messages",
	Short: "List/search emails",
	Long: `List emails with filtering and optional keyword search.

Examples:
  porteden email messages
  porteden email messages --unread
  porteden email messages --today
  porteden email messages --from boss@example.com
  porteden email messages -q "project update"
  porteden email messages --subject invoice --after 2026-02-01`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		params, err := buildEmailParams(cmd)
		if err != nil {
			return err
		}

		fetchAll, _ := cmd.Flags().GetBool("all")
		var response *api.EmailsResponse

		if fetchAll {
			response, err = client.GetAllEmails(params)
		} else {
			response, err = client.GetEmails(params)
		}
		if err != nil {
			return formatError(err)
		}

		output.PrintWithOptions(response, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

var messageCmd = &cobra.Command{
	Use:   "message <emailId>",
	Short: "Get a single email",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		emailID := args[0]
		includeBody, _ := cmd.Flags().GetBool("include-body")

		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		email, err := client.GetEmail(emailID, includeBody)
		if err != nil {
			return formatError(err)
		}

		output.PrintWithOptions(email, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

var threadCmd = &cobra.Command{
	Use:   "thread <threadId>",
	Short: "Get an email thread",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		threadID := args[0]

		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		thread, err := client.GetThread(threadID)
		if err != nil {
			return formatError(err)
		}

		output.PrintWithOptions(thread, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

var sendEmailCmd = &cobra.Command{
	Use:   "send",
	Short: "Send a new email",
	Long: `Send a new email.

Examples:
  porteden email send --to user@example.com --subject "Hello" --body "Hi there"
  porteden email send --to user@example.com --cc team@example.com --subject "Update" --body-file message.txt`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		req, err := buildSendEmailRequest(cmd)
		if err != nil {
			return err
		}

		resp, err := client.SendEmail(req)
		if err != nil {
			return formatError(err)
		}

		if resp.Success {
			fmt.Printf("Email sent successfully")
			if resp.EmailID != "" {
				fmt.Printf(" (ID: %s)", resp.EmailID)
			}
			fmt.Println()
		} else {
			return fmt.Errorf("failed to send email: %s", resp.ErrorMessage)
		}

		return nil
	},
}

var replyEmailCmd = &cobra.Command{
	Use:   "reply <emailId>",
	Short: "Reply to an email",
	Long: `Reply to an existing email.

Examples:
  porteden email reply <emailId> --body "Thanks for the update"
  porteden email reply <emailId> --body-file reply.txt --reply-all`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		emailID := args[0]

		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		req, err := buildReplyRequest(cmd)
		if err != nil {
			return err
		}

		resp, err := client.ReplyToEmail(emailID, req)
		if err != nil {
			return formatError(err)
		}

		if resp.Success {
			fmt.Printf("Reply sent successfully")
			if resp.EmailID != "" {
				fmt.Printf(" (ID: %s)", resp.EmailID)
			}
			fmt.Println()
		} else {
			return fmt.Errorf("failed to send reply: %s", resp.ErrorMessage)
		}

		return nil
	},
}

var forwardEmailCmd = &cobra.Command{
	Use:   "forward <emailId>",
	Short: "Forward an email",
	Long: `Forward an email to specified recipients.

Examples:
  porteden email forward <emailId> --to colleague@example.com
  porteden email forward <emailId> --to user@example.com --body "FYI"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		emailID := args[0]

		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		req, err := buildForwardRequest(cmd)
		if err != nil {
			return err
		}

		resp, err := client.ForwardEmail(emailID, req)
		if err != nil {
			return formatError(err)
		}

		if resp.Success {
			fmt.Printf("Email forwarded successfully")
			if resp.EmailID != "" {
				fmt.Printf(" (ID: %s)", resp.EmailID)
			}
			fmt.Println()
		} else {
			return fmt.Errorf("failed to forward email: %s", resp.ErrorMessage)
		}

		return nil
	},
}

var deleteEmailCmd = &cobra.Command{
	Use:   "delete <emailId>",
	Short: "Delete (trash) an email",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		emailID := args[0]

		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		if err := client.DeleteEmail(emailID); err != nil {
			return formatError(err)
		}

		fmt.Printf("Email deleted: %s\n", emailID)
		return nil
	},
}

var modifyEmailCmd = &cobra.Command{
	Use:   "modify <emailId>",
	Short: "Modify email properties",
	Long: `Modify email properties such as read status and labels.

Examples:
  porteden email modify <emailId> --mark-read
  porteden email modify <emailId> --mark-unread
  porteden email modify <emailId> --add-labels IMPORTANT,STARRED
  porteden email modify <emailId> --remove-labels INBOX`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		emailID := args[0]

		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		req, err := buildModifyRequest(cmd)
		if err != nil {
			return err
		}

		if err := client.ModifyEmail(emailID, req); err != nil {
			return formatError(err)
		}

		fmt.Printf("Email modified: %s\n", emailID)
		return nil
	},
}

func init() {
	// Messages command flags (search/filter)
	messagesCmd.Flags().StringP("query", "q", "", "Free-text search query")
	messagesCmd.Flags().String("from", "", "Filter by sender email")
	messagesCmd.Flags().String("to", "", "Filter by recipient email")
	messagesCmd.Flags().String("subject", "", "Filter by subject (partial match)")
	messagesCmd.Flags().String("label", "", "Filter by label/category")
	messagesCmd.Flags().Bool("unread", false, "Show only unread emails")
	messagesCmd.Flags().Bool("has-attachment", false, "Show only emails with attachments")
	messagesCmd.Flags().Int("limit", 20, "Maximum emails to return (1-50)")
	messagesCmd.Flags().Bool("include-body", false, "Include full email body in results")
	messagesCmd.Flags().Bool("all", false, "Fetch all pages")

	// Time filters for messages
	messagesCmd.Flags().Bool("today", false, "Show today's emails")
	messagesCmd.Flags().Bool("yesterday", false, "Show yesterday's emails")
	messagesCmd.Flags().Bool("week", false, "Show this week's emails")
	messagesCmd.Flags().Int("days", 0, "Show emails from the last N days")
	messagesCmd.Flags().String("after", "", "Emails after this date (YYYY-MM-DD or RFC3339)")
	messagesCmd.Flags().String("before", "", "Emails before this date (YYYY-MM-DD or RFC3339)")

	// Message command flags
	messageCmd.Flags().Bool("include-body", true, "Include full email body")

	// Send command flags
	sendEmailCmd.Flags().StringSlice("to", nil, "To recipients (email or Name <email> format)")
	sendEmailCmd.Flags().StringSlice("cc", nil, "CC recipients")
	sendEmailCmd.Flags().StringSlice("bcc", nil, "BCC recipients")
	sendEmailCmd.Flags().String("subject", "", "Email subject")
	sendEmailCmd.Flags().String("body", "", "Email body content")
	sendEmailCmd.Flags().String("body-file", "", "Read body from file")
	sendEmailCmd.Flags().String("body-type", "html", "Body type: html or text")
	sendEmailCmd.Flags().String("importance", "normal", "Importance: low, normal, high")
	sendEmailCmd.Flags().Int64("connection-id", 0, "Specific connection to send from")
	_ = sendEmailCmd.MarkFlagRequired("to")
	_ = sendEmailCmd.MarkFlagRequired("subject")

	// Reply command flags
	replyEmailCmd.Flags().String("body", "", "Reply body content")
	replyEmailCmd.Flags().String("body-file", "", "Read body from file")
	replyEmailCmd.Flags().String("body-type", "html", "Body type: html or text")
	replyEmailCmd.Flags().Bool("reply-all", false, "Reply to all recipients")

	// Forward command flags
	forwardEmailCmd.Flags().StringSlice("to", nil, "Forward recipients")
	forwardEmailCmd.Flags().StringSlice("cc", nil, "CC recipients")
	forwardEmailCmd.Flags().String("body", "", "Optional message to prepend")
	forwardEmailCmd.Flags().String("body-file", "", "Read body from file")
	forwardEmailCmd.Flags().String("body-type", "html", "Body type: html or text")
	_ = forwardEmailCmd.MarkFlagRequired("to")

	// Modify command flags
	modifyEmailCmd.Flags().Bool("mark-read", false, "Mark email as read")
	modifyEmailCmd.Flags().Bool("mark-unread", false, "Mark email as unread")
	modifyEmailCmd.Flags().StringSlice("add-labels", nil, "Labels to add")
	modifyEmailCmd.Flags().StringSlice("remove-labels", nil, "Labels to remove")

	// Register subcommands
	emailCmd.AddCommand(messagesCmd)
	emailCmd.AddCommand(messageCmd)
	emailCmd.AddCommand(threadCmd)
	emailCmd.AddCommand(sendEmailCmd)
	emailCmd.AddCommand(replyEmailCmd)
	emailCmd.AddCommand(forwardEmailCmd)
	emailCmd.AddCommand(deleteEmailCmd)
	emailCmd.AddCommand(modifyEmailCmd)
}

// buildEmailParams builds email search parameters from command flags
func buildEmailParams(cmd *cobra.Command) (api.EmailParams, error) {
	params := api.EmailParams{
		Limit: 20,
	}

	if query, _ := cmd.Flags().GetString("query"); query != "" {
		params.Query = query
	}
	if from, _ := cmd.Flags().GetString("from"); from != "" {
		params.From = from
	}
	if to, _ := cmd.Flags().GetString("to"); to != "" {
		params.To = to
	}
	if subject, _ := cmd.Flags().GetString("subject"); subject != "" {
		params.Subject = subject
	}
	if label, _ := cmd.Flags().GetString("label"); label != "" {
		params.Label = label
	}

	if cmd.Flags().Changed("unread") {
		unread, _ := cmd.Flags().GetBool("unread")
		params.Unread = &unread
	}
	if cmd.Flags().Changed("has-attachment") {
		hasAttachment, _ := cmd.Flags().GetBool("has-attachment")
		params.HasAttachment = &hasAttachment
	}

	if limit, _ := cmd.Flags().GetInt("limit"); limit > 0 {
		params.Limit = limit
	}

	if includeBody, _ := cmd.Flags().GetBool("include-body"); includeBody {
		params.IncludeBody = true
	}

	// Parse time range
	now := time.Now()
	today, _ := cmd.Flags().GetBool("today")
	yesterday, _ := cmd.Flags().GetBool("yesterday")
	week, _ := cmd.Flags().GetBool("week")
	days, _ := cmd.Flags().GetInt("days")
	afterStr, _ := cmd.Flags().GetString("after")
	beforeStr, _ := cmd.Flags().GetString("before")

	startOfDay := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	}

	if today {
		params.After = startOfDay(now)
		params.Before = params.After.AddDate(0, 0, 1)
	} else if yesterday {
		params.After = startOfDay(now.AddDate(0, 0, -1))
		params.Before = startOfDay(now)
	} else if week {
		params.After = startOfDay(now.AddDate(0, 0, -7))
		params.Before = startOfDay(now).AddDate(0, 0, 1)
	} else if days > 0 {
		params.After = startOfDay(now.AddDate(0, 0, -days))
		params.Before = startOfDay(now).AddDate(0, 0, 1)
	} else {
		if afterStr != "" {
			t, err := parseDateTime(afterStr)
			if err != nil {
				return params, fmt.Errorf("invalid after date: %w", err)
			}
			params.After = t
		}
		if beforeStr != "" {
			t, err := parseDateTime(beforeStr)
			if err != nil {
				return params, fmt.Errorf("invalid before date: %w", err)
			}
			params.Before = t
		}
	}

	return params, nil
}

// buildSendEmailRequest builds a send email request from command flags
func buildSendEmailRequest(cmd *cobra.Command) (api.SendEmailRequest, error) {
	req := api.SendEmailRequest{}

	toList, _ := cmd.Flags().GetStringSlice("to")
	for _, recipient := range toList {
		p := parseParticipant(recipient)
		req.To = append(req.To, p)
	}

	ccList, _ := cmd.Flags().GetStringSlice("cc")
	for _, recipient := range ccList {
		p := parseParticipant(recipient)
		req.CC = append(req.CC, p)
	}

	bccList, _ := cmd.Flags().GetStringSlice("bcc")
	for _, recipient := range bccList {
		p := parseParticipant(recipient)
		req.BCC = append(req.BCC, p)
	}

	req.Subject, _ = cmd.Flags().GetString("subject")

	body, err := getBodyContent(cmd)
	if err != nil {
		return req, err
	}
	if body == "" {
		return req, fmt.Errorf("either --body or --body-file is required")
	}
	req.Body = body

	req.BodyType, _ = cmd.Flags().GetString("body-type")

	importance, _ := cmd.Flags().GetString("importance")
	if importance != "" && importance != "normal" {
		req.Importance = importance
	}

	if cmd.Flags().Changed("connection-id") {
		connID, _ := cmd.Flags().GetInt64("connection-id")
		req.ConnectionID = &connID
	}

	return req, nil
}

// buildReplyRequest builds a reply request from command flags
func buildReplyRequest(cmd *cobra.Command) (api.ReplyEmailRequest, error) {
	req := api.ReplyEmailRequest{}

	body, err := getBodyContent(cmd)
	if err != nil {
		return req, err
	}
	if body == "" {
		return req, fmt.Errorf("either --body or --body-file is required")
	}
	req.Body = body
	req.BodyType, _ = cmd.Flags().GetString("body-type")
	req.ReplyAll, _ = cmd.Flags().GetBool("reply-all")

	return req, nil
}

// buildForwardRequest builds a forward request from command flags
func buildForwardRequest(cmd *cobra.Command) (api.ForwardEmailRequest, error) {
	req := api.ForwardEmailRequest{}

	toList, _ := cmd.Flags().GetStringSlice("to")
	for _, recipient := range toList {
		p := parseParticipant(recipient)
		req.To = append(req.To, p)
	}

	ccList, _ := cmd.Flags().GetStringSlice("cc")
	for _, recipient := range ccList {
		p := parseParticipant(recipient)
		req.CC = append(req.CC, p)
	}

	body, err := getBodyContent(cmd)
	if err != nil {
		return req, err
	}
	req.Body = body
	req.BodyType, _ = cmd.Flags().GetString("body-type")

	return req, nil
}

// buildModifyRequest builds a modify request from command flags
func buildModifyRequest(cmd *cobra.Command) (api.ModifyEmailRequest, error) {
	req := api.ModifyEmailRequest{}

	markRead := cmd.Flags().Changed("mark-read")
	markUnread := cmd.Flags().Changed("mark-unread")

	if markRead && markUnread {
		return req, fmt.Errorf("cannot use both --mark-read and --mark-unread")
	}

	if markRead {
		val := true
		req.MarkAsRead = &val
	} else if markUnread {
		val := false
		req.MarkAsRead = &val
	}

	if cmd.Flags().Changed("add-labels") {
		req.AddLabels, _ = cmd.Flags().GetStringSlice("add-labels")
	}

	if cmd.Flags().Changed("remove-labels") {
		req.RemoveLabels, _ = cmd.Flags().GetStringSlice("remove-labels")
	}

	if req.MarkAsRead == nil && len(req.AddLabels) == 0 && len(req.RemoveLabels) == 0 {
		return req, fmt.Errorf("at least one modification is required (--mark-read, --mark-unread, --add-labels, --remove-labels)")
	}

	return req, nil
}

// getBodyContent reads body content from --body flag or --body-file flag
func getBodyContent(cmd *cobra.Command) (string, error) {
	bodyStr, _ := cmd.Flags().GetString("body")
	bodyFile, _ := cmd.Flags().GetString("body-file")

	if bodyStr != "" && bodyFile != "" {
		return "", fmt.Errorf("cannot use both --body and --body-file")
	}

	if bodyFile != "" {
		content, err := os.ReadFile(bodyFile)
		if err != nil {
			return "", fmt.Errorf("failed to read body file: %w", err)
		}
		return string(content), nil
	}

	return bodyStr, nil
}

// parseParticipant parses a participant string.
// Supports formats: "email@example.com", "Name <email@example.com>", or "<email@example.com>"
func parseParticipant(s string) api.Participant {
	s = strings.TrimSpace(s)

	// Try "Name <email>" or "<email>" format
	if idx := strings.LastIndex(s, "<"); idx >= 0 {
		if end := strings.Index(s[idx:], ">"); end > 0 {
			name := strings.TrimSpace(s[:idx])
			email := s[idx+1 : idx+end]
			return api.Participant{Email: email, Name: name}
		}
	}

	// Plain email
	return api.Participant{Email: s}
}
