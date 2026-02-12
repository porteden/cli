package output

import "fmt"

const bannerWidth = 41

// PrintBanner displays the branded PortEden CLI header.
func PrintBanner() {
	top := "┌" + repeat("─", bannerWidth) + "┐"
	bot := "└" + repeat("─", bannerWidth) + "┘"
	blank := "│" + repeat(" ", bannerWidth) + "│"

	fmt.Println()
	fmt.Println(ColorCyan(top))
	fmt.Println(ColorCyan(blank))
	fmt.Println(ColorCyan("│") + ColorBold(center("P O R T E D E N . C O M", bannerWidth)) + ColorCyan("│"))
	fmt.Println(ColorCyan("│") + ColorGray(center("CLI Setup", bannerWidth)) + ColorCyan("│"))
	fmt.Println(ColorCyan(blank))
	fmt.Println(ColorCyan("│") + center("Your data. Your rules.", bannerWidth) + ColorCyan("│"))
	fmt.Println(ColorCyan(blank))
	fmt.Println(ColorCyan(bot))
	fmt.Println()
}

// PrintStep prints a numbered wizard step, e.g. "[1/3] Opening browser..."
func PrintStep(n, total int, msg string) {
	prefix := fmt.Sprintf("[%d/%d]", n, total)
	fmt.Printf("  %s %s\n", ColorCyan(prefix), msg)
}

// PrintSuccess prints a green checkmark line.
func PrintSuccess(msg string) {
	fmt.Printf("  %s %s\n", ColorGreen("✓"), msg)
}

// PrintInfo prints an indented gray info line.
func PrintInfo(msg string) {
	fmt.Printf("        %s\n", ColorGray(msg))
}

// PrintDivider prints a thin separator line.
func PrintDivider() {
	fmt.Println()
	fmt.Println(ColorGray(repeat("─", bannerWidth+2)))
	fmt.Println()
}

// PrintCompletion prints the final success block with quick-start hints.
func PrintCompletion(profile string) {
	PrintDivider()
	PrintSuccess(ColorBold("You're all set!"))
	fmt.Printf("  Profile: %s\n", ColorCyan(profile))
	fmt.Println()
	fmt.Println(ColorBold("  Get started:"))
	fmt.Printf("    %s        %s\n", ColorCyan("porteden calendar list"), "List your calendars")
	fmt.Printf("    %s       %s\n", ColorCyan("porteden events --today"), "Today's events")
	fmt.Printf("    %s          %s\n", ColorCyan("porteden auth status"), "Check connection")
	fmt.Println()
	fmt.Printf("  Need help? Check out the docs at %s\n", ColorCyan("https://docs.porteden.com/cli"))
	fmt.Println()
}

func center(s string, width int) string {
	if len(s) >= width {
		return s
	}
	pad := width - len(s)
	left := pad / 2
	right := pad - left
	return repeat(" ", left) + s + repeat(" ", right)
}

func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
