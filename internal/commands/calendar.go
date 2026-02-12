package commands

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/porteden/cli/internal/api"
	"github.com/porteden/cli/internal/apierr"
	"github.com/porteden/cli/internal/auth"
	"github.com/porteden/cli/internal/output"
	"github.com/spf13/cobra"
)

var calendarCmd = &cobra.Command{
	Use:     "calendar",
	Short:   "Calendar commands",
	Aliases: []string{"cal"},
}

var calendarsCmd = &cobra.Command{
	Use:   "calendars",
	Short: "List calendars",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		calendars, err := client.GetCalendars()
		if err != nil {
			return formatError(err)
		}

		output.PrintWithOptions(calendars, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "List/search events",
	Long: `List calendar events with time filtering and optional keyword search.

Examples:
  porteden calendar events --today
  porteden calendar events --tomorrow
  porteden calendar events --week
  porteden calendar events --days 7
  porteden calendar events --from 2026-02-01 --to 2026-02-28
  porteden calendar events -q "budget review"
  porteden calendar events -q "meeting" --attendees "finance@example.com,cfo@example.com"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		params, err := buildEventParams(cmd)
		if err != nil {
			return err
		}

		fetchAll, _ := cmd.Flags().GetBool("all")
		var events *api.EventsResponse

		if fetchAll {
			events, err = client.GetAllEvents(params)
		} else {
			events, err = client.GetEvents(params)
		}
		if err != nil {
			return formatError(err)
		}

		output.PrintWithOptions(events, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

var eventCmd = &cobra.Command{
	Use:   "event <eventId>",
	Short: "Get a single event",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eventID := args[0]
		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		resp, err := client.GetEvent(eventID)
		if err != nil {
			return formatError(err)
		}

		output.PrintWithOptions(resp, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an event",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		calendarID, _ := cmd.Flags().GetInt64("calendar")
		summary, _ := cmd.Flags().GetString("summary")
		fromStr, _ := cmd.Flags().GetString("from")
		toStr, _ := cmd.Flags().GetString("to")
		description, _ := cmd.Flags().GetString("description")
		location, _ := cmd.Flags().GetString("location")
		attendees, _ := cmd.Flags().GetStringSlice("attendees")
		allDay, _ := cmd.Flags().GetBool("all-day")
		recurrence, _ := cmd.Flags().GetStringSlice("recurrence")

		// Parse times
		startTime, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return fmt.Errorf("invalid start time: %w", err)
		}

		endTime, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			return fmt.Errorf("invalid end time: %w", err)
		}

		req := api.CreateEventRequest{
			CalendarID:  calendarID,
			Summary:     summary,
			Description: description,
			Location:    location,
			From:        startTime,
			To:          endTime,
			IsAllDay:    allDay,
			Attendees:   attendees,
			Recurrence:  recurrence,
		}

		event, err := client.CreateEvent(req)
		if err != nil {
			return formatError(err)
		}

		fmt.Printf("Event created successfully (ID: %s)\n", event.ID)
		output.PrintWithOptions(event, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

var updateCmd = &cobra.Command{
	Use:   "update <eventId>",
	Short: "Update an existing event",
	Long: `Update an existing calendar event. All fields are optional.

Examples:
  porteden calendar update <eventId> --summary "New Title"
  porteden calendar update <eventId> --from "2026-02-10T10:00:00Z" --to "2026-02-10T11:00:00Z"
  porteden calendar update <eventId> --add-attendees "new@example.com"
  porteden calendar update <eventId> --remove-attendees "old@example.com" --notify`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eventID := args[0]
		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		req := api.UpdateEventRequest{}

		if cmd.Flags().Changed("summary") {
			req.Summary, _ = cmd.Flags().GetString("summary")
		}
		if cmd.Flags().Changed("description") {
			req.Description, _ = cmd.Flags().GetString("description")
		}
		if cmd.Flags().Changed("location") {
			req.Location, _ = cmd.Flags().GetString("location")
		}
		if cmd.Flags().Changed("from") {
			fromStr, _ := cmd.Flags().GetString("from")
			t, err := time.Parse(time.RFC3339, fromStr)
			if err != nil {
				return fmt.Errorf("invalid start time: %w", err)
			}
			req.From = &t
		}
		if cmd.Flags().Changed("to") {
			toStr, _ := cmd.Flags().GetString("to")
			t, err := time.Parse(time.RFC3339, toStr)
			if err != nil {
				return fmt.Errorf("invalid end time: %w", err)
			}
			req.To = &t
		}
		if cmd.Flags().Changed("all-day") {
			allDay, _ := cmd.Flags().GetBool("all-day")
			req.IsAllDay = &allDay
		}
		if cmd.Flags().Changed("add-attendees") {
			req.AddAttendees, _ = cmd.Flags().GetStringSlice("add-attendees")
		}
		if cmd.Flags().Changed("remove-attendees") {
			req.RemoveAttendees, _ = cmd.Flags().GetStringSlice("remove-attendees")
		}
		if cmd.Flags().Changed("notify") {
			notify, _ := cmd.Flags().GetBool("notify")
			req.SendNotifications = &notify
		}

		event, err := client.UpdateEvent(eventID, req)
		if err != nil {
			return formatError(err)
		}

		fmt.Printf("Event updated successfully (ID: %s)\n", event.ID)
		output.PrintWithOptions(event, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete <eventId>",
	Short: "Delete an event",
	Long: `Delete a calendar event.

Examples:
  porteden calendar delete <eventId>
  porteden calendar delete <eventId> --no-notify`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eventID := args[0]
		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		noNotify, _ := cmd.Flags().GetBool("no-notify")
		notifyAttendees := !noNotify

		resp, err := client.DeleteEvent(eventID, notifyAttendees)
		if err != nil {
			return formatError(err)
		}

		fmt.Printf("Event deleted: %s\n", resp.Message)
		return nil
	},
}

var respondCmd = &cobra.Command{
	Use:   "respond <eventId> <status>",
	Short: "Respond to an event invitation",
	Long: `Respond to an event invitation with one of:
  - accepted
  - declined
  - tentative`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		eventID := args[0]
		status := args[1]

		// Validate status
		validStatuses := map[string]bool{
			"accepted":  true,
			"declined":  true,
			"tentative": true,
		}
		if !validStatuses[status] {
			return fmt.Errorf("invalid status: %s (must be accepted, declined, or tentative)", status)
		}

		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		event, err := client.RespondToEvent(eventID, status)
		if err != nil {
			return formatError(err)
		}

		fmt.Printf("Response recorded: %s\n", status)
		output.PrintWithOptions(event, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

var freebusyCmd = &cobra.Command{
	Use:   "freebusy",
	Short: "Get free/busy information",
	Long: `Get free/busy time blocks for calendars.

Examples:
  porteden calendar freebusy --today
  porteden calendar freebusy --week
  porteden calendar freebusy --from 2026-02-05 --to 2026-02-12
  porteden calendar freebusy --week --calendars 123,456`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		// Reuse buildEventParams for time range parsing
		eventParams, err := buildEventParams(cmd)
		if err != nil {
			return err
		}

		calendars, _ := cmd.Flags().GetString("calendars")

		params := api.FreeBusyParams{
			From:      eventParams.From,
			To:        eventParams.To,
			Calendars: calendars,
		}

		resp, err := client.GetFreeBusy(params)
		if err != nil {
			return formatError(err)
		}

		output.PrintWithOptions(resp, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

var byContactCmd = &cobra.Command{
	Use:   "by-contact [email]",
	Short: "List events with a specific contact",
	Long: `List calendar events that include a specific contact as an attendee.
At least one of email or --name is required.
Email and name parameters support partial matching (case-insensitive).

Examples:
  porteden calendar by-contact user@example.com
  porteden calendar by-contact --name "John"
  porteden calendar by-contact --name "Smith" --email "@acme.com"`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		contactEmail := ""
		if len(args) > 0 {
			contactEmail = args[0]
		}

		contactName, _ := cmd.Flags().GetString("name")

		// Validate at least one identifier is provided
		if contactEmail == "" && contactName == "" {
			return fmt.Errorf("at least one of email or --name is required")
		}

		client, err := getClient(cmd)
		if err != nil {
			return err
		}

		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")

		params := api.EventsByContactParams{
			Email:  contactEmail,
			Name:   contactName,
			Limit:  limit,
			Offset: offset,
		}

		fetchAll, _ := cmd.Flags().GetBool("all")
		var events *api.EventsResponse

		if fetchAll {
			events, err = getAllEventsByContact(client, params)
		} else {
			events, err = client.GetEventsByContact(params)
		}
		if err != nil {
			return formatError(err)
		}

		output.PrintWithOptions(events, getOutputFormat(cmd), output.PrintOptions{
			Compact: IsCompactMode(),
		})
		return nil
	},
}

// getAllEventsByContact fetches all events by contact by auto-paginating
func getAllEventsByContact(client *api.Client, params api.EventsByContactParams) (*api.EventsResponse, error) {
	var allEvents []api.Event
	offset := 0
	var accessInfo string
	var calEmail string

	for {
		params.Offset = offset
		resp, err := client.GetEventsByContact(params)
		if err != nil {
			return nil, err
		}

		allEvents = append(allEvents, resp.Events...)
		accessInfo = resp.AccessInfo
		calEmail = resp.CurrentUserCalendarEmail

		if resp.Meta == nil || !resp.Meta.HasMore {
			finalMeta := &api.Meta{
				Count:      len(allEvents),
				TotalCount: len(allEvents),
			}
			if resp.Meta != nil {
				finalMeta.From = resp.Meta.From
				finalMeta.To = resp.Meta.To
				finalMeta.Timestamp = resp.Meta.Timestamp
			}
			return &api.EventsResponse{
				RequestID:                resp.RequestID,
				Events:                   allEvents,
				Meta:                     finalMeta,
				AccessInfo:               accessInfo,
				CurrentUserCalendarEmail: calEmail,
			}, nil
		}

		offset += resp.Meta.Count
	}
}

func init() {
	// Time filter flags (used by events and freebusy)
	for _, cmd := range []*cobra.Command{eventsCmd, freebusyCmd} {
		cmd.Flags().Bool("today", false, "Show today's events")
		cmd.Flags().Bool("tomorrow", false, "Show tomorrow's events")
		cmd.Flags().Bool("week", false, "Show this week's events")
		cmd.Flags().Int("days", 0, "Show events for the next N days")
		cmd.Flags().String("from", "", "Start date (YYYY-MM-DD or datetime)")
		cmd.Flags().String("to", "", "End date (YYYY-MM-DD or datetime)")
		cmd.Flags().Int("limit", 50, "Maximum events to return")
		cmd.Flags().Int("offset", 0, "Skip first N events (pagination)")
		cmd.Flags().Bool("all", false, "Fetch all pages")
	}

	// Events-specific flags
	eventsCmd.Flags().Int64("calendar", 0, "Filter by calendar ID")
	eventsCmd.Flags().Bool("include-cancelled", false, "Include cancelled events (default: false)")
	eventsCmd.Flags().StringP("query", "q", "", "Keyword search in title, description, location")
	eventsCmd.Flags().String("attendees", "", "Comma-separated attendee emails to filter by")

	// Freebusy-specific flags
	freebusyCmd.Flags().String("calendars", "", "Comma-separated calendar IDs")

	// By-contact flags (no time filters in v2 API)
	byContactCmd.Flags().String("name", "", "Filter by contact name (partial match, case-insensitive)")
	byContactCmd.Flags().Int("limit", 50, "Maximum events to return")
	byContactCmd.Flags().Int("offset", 0, "Skip first N events (pagination)")
	byContactCmd.Flags().Bool("all", false, "Fetch all pages")

	// Create flags
	createCmd.Flags().Int64("calendar", 0, "Calendar ID (required)")
	createCmd.Flags().String("summary", "", "Event title (required)")
	createCmd.Flags().String("from", "", "Start time (required)")
	createCmd.Flags().String("to", "", "End time (required)")
	createCmd.Flags().String("description", "", "Event description")
	createCmd.Flags().String("location", "", "Event location")
	createCmd.Flags().StringSlice("attendees", nil, "Attendee emails")
	createCmd.Flags().Bool("all-day", false, "Create all-day event")
	createCmd.Flags().StringSlice("recurrence", nil, "RRULE recurrence patterns")
	_ = createCmd.MarkFlagRequired("calendar")
	_ = createCmd.MarkFlagRequired("summary")
	_ = createCmd.MarkFlagRequired("from")
	_ = createCmd.MarkFlagRequired("to")

	// Update flags
	updateCmd.Flags().String("summary", "", "New event title")
	updateCmd.Flags().String("description", "", "New description")
	updateCmd.Flags().String("location", "", "New location")
	updateCmd.Flags().String("from", "", "New start time (RFC3339)")
	updateCmd.Flags().String("to", "", "New end time (RFC3339)")
	updateCmd.Flags().Bool("all-day", false, "Set as all-day event")
	updateCmd.Flags().StringSlice("add-attendees", nil, "Emails to add as attendees")
	updateCmd.Flags().StringSlice("remove-attendees", nil, "Emails to remove from attendees")
	updateCmd.Flags().Bool("notify", true, "Send notifications to attendees")

	// Delete flags
	deleteCmd.Flags().Bool("no-notify", false, "Don't send cancellation notifications")

	calendarCmd.AddCommand(calendarsCmd)
	calendarCmd.AddCommand(eventsCmd)
	calendarCmd.AddCommand(eventCmd)
	calendarCmd.AddCommand(createCmd)
	calendarCmd.AddCommand(updateCmd)
	calendarCmd.AddCommand(deleteCmd)
	calendarCmd.AddCommand(respondCmd)
	calendarCmd.AddCommand(byContactCmd)
	calendarCmd.AddCommand(freebusyCmd)
}

// Helper function to get API client.
// If not authenticated and running in an interactive terminal, offers to run the setup wizard.
func getClient(cmd *cobra.Command) (*api.Client, error) {
	profileName := getProfile(cmd)
	apiKey, err := auth.GetAPIKey(profileName)
	if err == nil {
		return api.NewClient(apiKey), nil
	}

	// Non-interactive: return plain error
	if !auth.IsInteractiveTerminal() {
		return nil, fmt.Errorf("not authenticated. Run 'porteden auth login' to authenticate")
	}

	// Interactive: offer setup wizard
	output.PrintBanner()
	fmt.Println("  No account configured yet.")
	fmt.Print("  Would you like to set up now? " + output.ColorGray("[Y/n]: "))

	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	choice := strings.TrimSpace(strings.ToLower(line))
	if choice != "" && choice != "y" && choice != "yes" {
		fmt.Println()
		return nil, fmt.Errorf("not authenticated. Run 'porteden auth login' to authenticate")
	}
	fmt.Println()

	// Initialize credential store for the wizard
	if err := auth.InitStore(); err != nil {
		return nil, err
	}

	wizardKey, err := runLoginWizard(profileName, "")
	if err != nil {
		return nil, err
	}

	return api.NewClient(wizardKey), nil
}

// Helper function to build event parameters from flags
func buildEventParams(cmd *cobra.Command) (api.EventParams, error) {
	params := api.EventParams{
		Limit: 50,
	}

	// Get limit
	if limit, _ := cmd.Flags().GetInt("limit"); limit > 0 {
		params.Limit = limit
	}

	// Get offset
	if offset, _ := cmd.Flags().GetInt("offset"); offset > 0 {
		params.Offset = offset
	}

	// Get calendar ID (only supported by events endpoint)
	if cmd.Flags().Changed("calendar") {
		if calID, _ := cmd.Flags().GetInt64("calendar"); calID > 0 {
			params.CalendarID = calID
		}
	}

	// Get includeCancelled (only for events endpoint)
	if cmd.Flags().Changed("include-cancelled") {
		params.IncludeCancelled, _ = cmd.Flags().GetBool("include-cancelled")
	}

	// Get query (for keyword search via events endpoint)
	if query, _ := cmd.Flags().GetString("query"); query != "" {
		params.Query = query
	}

	// Get attendees filter
	if attendees, _ := cmd.Flags().GetString("attendees"); attendees != "" {
		params.Attendees = attendees
	}

	// Parse time range
	now := time.Now()
	today, _ := cmd.Flags().GetBool("today")
	tomorrow, _ := cmd.Flags().GetBool("tomorrow")
	week, _ := cmd.Flags().GetBool("week")
	days, _ := cmd.Flags().GetInt("days")
	fromStr, _ := cmd.Flags().GetString("from")
	toStr, _ := cmd.Flags().GetString("to")

	// Helper to get start of day
	startOfDay := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	}

	if today {
		params.From = startOfDay(now)
		params.To = params.From.AddDate(0, 0, 1)
	} else if tomorrow {
		params.From = startOfDay(now.AddDate(0, 0, 1))
		params.To = params.From.AddDate(0, 0, 1)
	} else if week {
		params.From = startOfDay(now)
		params.To = params.From.AddDate(0, 0, 7)
	} else if days > 0 {
		params.From = startOfDay(now)
		params.To = params.From.AddDate(0, 0, days)
	} else if fromStr != "" && toStr != "" {
		var err error
		params.From, err = parseDateTime(fromStr)
		if err != nil {
			return params, fmt.Errorf("invalid from date: %w", err)
		}
		params.To, err = parseDateTime(toStr)
		if err != nil {
			return params, fmt.Errorf("invalid to date: %w", err)
		}
	} else {
		// Default: next 7 days
		params.From = startOfDay(now)
		params.To = params.From.AddDate(0, 0, 7)
	}

	return params, nil
}

// Helper function to parse date/datetime strings
func parseDateTime(s string) (time.Time, error) {
	// Try RFC3339 first (full datetime)
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	// Try date only (YYYY-MM-DD)
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("invalid date format (use YYYY-MM-DD or RFC3339)")
}

// Helper function to format API errors
func formatError(err error) error {
	if apiErr, ok := err.(*apierr.APIError); ok {
		return errors.New(apierr.UserFriendlyError(apiErr))
	}
	return err
}
