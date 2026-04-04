package cmd

import (
	"fmt"
	"html"
	"log"
	"mairu/internal/agent"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	tele "gopkg.in/telebot.v3"
)

func formatTelegramHTML(md string) string {
	res := html.EscapeString(md)

	// Code blocks
	reCodeBlock := regexp.MustCompile(`(?s)` + "```" + `(?:[a-zA-Z0-9]+)?\n?(.*?)` + "```")
	res = reCodeBlock.ReplaceAllString(res, "<pre><code>$1</code></pre>")

	// Inline code
	reInlineCode := regexp.MustCompile("`([^`]+)`")
	res = reInlineCode.ReplaceAllString(res, "<code>$1</code>")

	// Bold
	reBold := regexp.MustCompile(`\*\*(.*?)\*\*`)
	res = reBold.ReplaceAllString(res, "<b>$1</b>")

	// Italic
	reItalic := regexp.MustCompile(`\*([^\*]+)\*`)
	res = reItalic.ReplaceAllString(res, "<i>$1</i>")

	return res
}

func sendLongMessage(c tele.Context, text string) error {
	lines := strings.Split(text, "\n")
	var chunk string

	sendChunk := func(msg string) error {
		msg = strings.TrimSpace(msg)
		if msg == "" {
			return nil
		}
		htmlMsg := formatTelegramHTML(msg)
		err := c.Send(htmlMsg, &tele.SendOptions{ParseMode: tele.ModeHTML})
		if err != nil {
			log.Printf("HTML send failed, falling back to plain text: %v", err)
			return c.Send(msg)
		}
		return nil
	}

	for _, line := range lines {
		for len(line) > 4000 {
			if len(chunk) > 0 {
				sendChunk(chunk)
				chunk = ""
			}
			sendChunk(line[:4000])
			line = line[4000:]
		}

		if len(chunk)+len(line)+1 > 4000 {
			sendChunk(chunk)
			chunk = line + "\n"
		} else {
			chunk += line + "\n"
		}
	}

	if chunk != "" {
		sendChunk(chunk)
	}
	return nil
}

var telegramCmd = &cobra.Command{
	Use:   "telegram",
	Short: "Start Telegram bot interface",
	Run: func(cmd *cobra.Command, args []string) {
		token := os.Getenv("TELEGRAM_BOT_TOKEN")
		if token == "" {
			log.Fatal("TELEGRAM_BOT_TOKEN environment variable is required")
		}

		apiKey := GetAPIKey()
		if apiKey == "" {
			log.Fatal("GEMINI_API_KEY environment variable is required")
		}

		projectRoot, _ := cmd.Flags().GetString("project")
		if projectRoot == "" {
			pwd, err := os.Getwd()
			if err != nil {
				log.Fatalf("failed to get current directory: %v", err)
			}
			projectRoot = pwd
		}

		allowedUsersRaw, _ := cmd.Flags().GetString("allowed-users")
		if allowedUsersRaw == "" {
			allowedUsersRaw = os.Getenv("TELEGRAM_ALLOWED_USERS")
		}

		allowedUsers := make(map[string]bool)
		for _, u := range strings.Split(allowedUsersRaw, ",") {
			u = strings.TrimSpace(u)
			if u != "" {
				allowedUsers[u] = true
			}
		}

		meiliURL, _ := cmd.Flags().GetString("meili-url")
		meiliAPIKey, _ := cmd.Flags().GetString("meili-api-key")

		pref := tele.Settings{
			Token:  token,
			Poller: &tele.LongPoller{Timeout: 10 * time.Second},
		}

		b, err := tele.NewBot(pref)
		if err != nil {
			log.Fatalf("failed to create telegram bot: %v", err)
		}

		b.Handle("/help", func(c tele.Context) error {
			helpText := `<b>Available Commands:</b>
- /help: Show this help message
- /clear: Clear the terminal context and session
- /save: Save the current session
- /compact: Compact the current session history
- !cmd: Run a local bash command and append its output to your prompt
- !!cmd: Run a local bash command immediately (output returned to you)
- @file/path: Include file contents in your prompt`
			return c.Send(helpText, &tele.SendOptions{ParseMode: tele.ModeHTML})
		})

		b.Handle("/clear", func(c tele.Context) error {
			log.Printf("Received /clear from %d", c.Sender().ID)
			sender := c.Sender()
			senderID := strconv.FormatInt(sender.ID, 10)
			senderUsername := sender.Username

			if len(allowedUsers) > 0 {
				if !allowedUsers[senderID] && !allowedUsers[senderUsername] {
					return c.Send("Unauthorized.")
				}
			}

			sessionName := fmt.Sprintf("tg-%d", c.Chat().ID)
			ag, err := agent.New(projectRoot, apiKey, agent.Config{
				MeiliURL:    meiliURL,
				MeiliAPIKey: meiliAPIKey,
			})
			if err != nil {
				return c.Send("Error initializing agent.")
			}
			defer ag.Close()

			ag.ResetSession()
			if err := ag.SaveSession(sessionName); err != nil {
				return c.Send("Failed to clear session.")
			}
			return c.Send("Context cleared.")
		})

		b.Handle("/save", func(c tele.Context) error {
			sessionName := fmt.Sprintf("tg-%d", c.Chat().ID)
			ag, err := agent.New(projectRoot, apiKey, agent.Config{
				MeiliURL:    meiliURL,
				MeiliAPIKey: meiliAPIKey,
			})
			if err != nil {
				return c.Send("Error initializing agent.")
			}
			defer ag.Close()

			if err := ag.SaveSession(sessionName); err != nil {
				return c.Send("Failed to save session: " + err.Error())
			}
			return c.Send("Session saved.")
		})

		b.Handle("/compact", func(c tele.Context) error {
			sessionName := fmt.Sprintf("tg-%d", c.Chat().ID)
			ag, err := agent.New(projectRoot, apiKey, agent.Config{
				MeiliURL:    meiliURL,
				MeiliAPIKey: meiliAPIKey,
			})
			if err != nil {
				return c.Send("Error initializing agent.")
			}
			defer ag.Close()

			if err := ag.LoadSession(sessionName); err != nil {
				return c.Send("Failed to load session.")
			}

			if err := ag.CompactContext(); err != nil {
				return c.Send("Failed to compact context: " + err.Error())
			}

			if err := ag.SaveSession(sessionName); err != nil {
				return c.Send("Failed to save compacted session.")
			}
			return c.Send("Session context compacted successfully.")
		})

		b.Handle(tele.OnText, func(c tele.Context) error {
			sender := c.Sender()
			senderID := strconv.FormatInt(sender.ID, 10)
			senderUsername := sender.Username

			log.Printf("Received message from %s (ID: %s): %s", senderUsername, senderID, c.Text())

			if len(allowedUsers) > 0 {
				if !allowedUsers[senderID] && !allowedUsers[senderUsername] {
					log.Printf("Unauthorized access attempt by %s (ID: %s)", senderUsername, senderID)
					return c.Send("Unauthorized. Your user ID is " + senderID)
				}
			}

			// Map chat ID to an agent session
			sessionName := fmt.Sprintf("tg-%d", c.Chat().ID)

			ag, err := agent.New(projectRoot, apiKey, agent.Config{
				MeiliURL:    meiliURL,
				MeiliAPIKey: meiliAPIKey,
			})
			if err != nil {
				log.Printf("Failed to init agent: %v", err)
				return c.Send("Error initializing agent.")
			}
			defer ag.Close()

			if err := ag.LoadSession(sessionName); err != nil {
				log.Printf("Failed to load session: %v", err)
				return c.Send("Error loading session.")
			}

			// Notify user we are thinking
			_ = c.Notify(tele.Typing)

			prompt := c.Text()

			// Handle '!!cmd' - local immediate execute
			if strings.HasPrefix(prompt, "!!") {
				cmdStr := strings.TrimSpace(strings.TrimPrefix(prompt, "!!"))
				c.Send(fmt.Sprintf("<i>Running local command: %s</i>", html.EscapeString(cmdStr)), &tele.SendOptions{ParseMode: tele.ModeHTML})

				out, err := ag.RunBash(cmdStr, 60000)
				if err != nil {
					return sendLongMessage(c, fmt.Sprintf("❌ Command failed: %v\n<pre><code>%s</code></pre>", err, html.EscapeString(out)))
				}
				return sendLongMessage(c, fmt.Sprintf("<pre><code>%s</code></pre>", html.EscapeString(out)))
			}

			// Handle '!cmd' - run and append to prompt
			if strings.HasPrefix(prompt, "!") {
				cmdStr := strings.TrimSpace(strings.TrimPrefix(prompt, "!"))
				c.Send(fmt.Sprintf("<i>Running command to include in prompt: %s</i>", html.EscapeString(cmdStr)), &tele.SendOptions{ParseMode: tele.ModeHTML})

				out, err := ag.RunBash(cmdStr, 60000)
				if err != nil {
					prompt += fmt.Sprintf("\n\nCommand `!%s` failed: %v\nOutput: %s", cmdStr, err, out)
				} else {
					prompt += fmt.Sprintf("\n\nOutput of `!%s`:\n```\n%s\n```", cmdStr, out)
				}
			}

			// Handle '@file' - attach file to prompt
			fileRegex := regexp.MustCompile(`@([a-zA-Z0-9_./-]+)`)
			matches := fileRegex.FindAllStringSubmatch(prompt, -1)
			if len(matches) > 0 {
				for _, match := range matches {
					filePath := match[1]
					var content []byte
					var err error

					// Try from agent root first
					if ag != nil {
						content, err = os.ReadFile(fmt.Sprintf("%s/%s", ag.GetRoot(), filePath))
					}

					// Fallback to absolute or current working dir relative
					if err != nil {
						content, err = os.ReadFile(filePath)
					}

					if err == nil {
						prompt += fmt.Sprintf("\n\nFile: %s\n```\n%s\n```", filePath, string(content))
						c.Send(fmt.Sprintf("<i>Attached file: %s</i>", html.EscapeString(filePath)), &tele.SendOptions{ParseMode: tele.ModeHTML})
					} else {
						c.Send(fmt.Sprintf("❌ Could not read file @%s: %v", html.EscapeString(filePath), err), &tele.SendOptions{ParseMode: tele.ModeHTML})
					}
				}
			}

			outChan := make(chan agent.AgentEvent)
			go ag.RunStream(prompt, outChan)

			statusMsg, _ := c.Bot().Send(c.Chat(), "<i>Thinking...</i>", &tele.SendOptions{ParseMode: tele.ModeHTML})

			var textChunk strings.Builder
			var statusLog []string
			lastEdit := time.Now()

			for ev := range outChan {
				switch ev.Type {
				case "text":
					textChunk.WriteString(ev.Content)
				case "tool_call":
					_ = c.Notify(tele.Typing)
					statusLog = append(statusLog, fmt.Sprintf("🔧 <b>%s</b>", html.EscapeString(ev.ToolName)))
				case "status":
					statusLog = append(statusLog, fmt.Sprintf("ℹ️ %s", html.EscapeString(ev.Content)))
				case "diff":
					statusLog = append(statusLog, fmt.Sprintf("📝 <i>Applied edit</i>"))
					// Send diff immediately so user sees the code
					sendLongMessage(c, ev.Content)
				case "error":
					statusLog = append(statusLog, fmt.Sprintf("❌ %s", html.EscapeString(ev.Content)))
				}

				if len(statusLog) > 0 && time.Since(lastEdit) > time.Second && statusMsg != nil {
					display := statusLog
					if len(display) > 8 {
						display = display[len(display)-8:] // keep last 8 statuses
					}
					c.Bot().Edit(statusMsg, strings.Join(display, "\n"), &tele.SendOptions{ParseMode: tele.ModeHTML})
					lastEdit = time.Now()
				}
			}

			if statusMsg != nil {
				display := statusLog
				if len(display) > 8 {
					display = display[len(display)-8:]
				}
				finalDisplay := strings.Join(display, "\n")
				if finalDisplay != "" {
					finalDisplay += "\n✅ <b>Finished operations.</b>"
					c.Bot().Edit(statusMsg, finalDisplay, &tele.SendOptions{ParseMode: tele.ModeHTML})
				} else {
					c.Bot().Delete(statusMsg)
				}
			}

			if err := ag.SaveSession(sessionName); err != nil {
				log.Printf("Failed to save session: %v", err)
			}

			finalText := textChunk.String()
			if finalText == "" && len(statusLog) == 0 {
				finalText = "Done."
			}

			if finalText != "" {
				return sendLongMessage(c, finalText)
			}
			return nil
		})

		fmt.Println("Telegram bot is running...")
		b.Start()
	},
}

func init() {
	telegramCmd.Flags().String("meili-url", os.Getenv("MEILI_URL"), "Meilisearch URL")
	telegramCmd.Flags().String("meili-api-key", os.Getenv("MEILI_API_KEY"), "Meilisearch API key")
	telegramCmd.Flags().StringP("project", "P", "", "Project root path (default is current directory)")
	telegramCmd.Flags().String("allowed-users", "", "Comma separated list of allowed telegram user IDs or usernames")
	rootCmd.AddCommand(telegramCmd)
}
