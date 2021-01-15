package main

import (
	"os"
	"fmt"
	"flag"
	"html"
	"strconv"
	"os/user"
	"encoding/json"
	"github.com/andygrunwald/go-jira"
	"github.com/davecgh/go-spew/spew"
	"github.com/fatih/color"
)

type Config struct {
	User string `json:"user"`
	Pass string `json:"pass"`
	Host string `json:"host"`
}

var config Config
var userStats = map[string][]int{}
var statusMap = map[string]int{
	"Новая": 0,
	"В Работе": 1,
	"Тестирование": 2,
	"Сделана": 3,
	"Закрыта": 4,
}

var smiles = map[string]string{
	"Scream": html.UnescapeString("&#128561;"),
	"New": html.UnescapeString("&#10024;"),
	"Done": html.UnescapeString("&#127937;"),
	"Work": html.UnescapeString("&#128296;"),
	"Test": html.UnescapeString("&#129514;"),
	"Damn": html.UnescapeString("&#129324;"),
	"Block": html.UnescapeString("&#128679;"),
	"Review": html.UnescapeString("&#128065;"),
	"Paused": html.UnescapeString("&#128012;"),
	"Closed": html.UnescapeString("&#129351;"),
	"Блокирующий": html.UnescapeString("&#9940;"),
	"Критический": html.UnescapeString("&#128293;"),
	"Высокий": html.UnescapeString("&#10071;"),
	"Обычный": html.UnescapeString("&#128317;"),
	"Не критичный": html.UnescapeString("&#128317;"),
	"Влияющий": html.UnescapeString("&#9196;"),
	"Низкий": html.UnescapeString("&#9196;"),
}

var activeStatus = map[string]bool{
	"Новая": true,
	"В Работе": true,
	"Тестирование": true,
}

var statusAlias = map[string]string{
	"Новая": "New",
	"В Работе": "Work",
	"Тестирование": "Test",
	"Сделана": "Done",
	"Ожидание": "Paused",
	"In Review": "Review",
	"Закрыта": "Closed",
}

var statusColourMap = map[string]*color.Color {
	"Новая": color.New(color.FgGreen),
	"В Работе": color.New(color.FgYellow),
	"Тестирование": color.New(color.FgMagenta),
	"Сделана": color.New(color.FgBlue),
	"Закрыта": color.New(color.FgRed),
}

var debug bool
var active bool
var nocolor bool
var links bool
var markup bool

var allIssues = map[string]map[string]bool{}

func main() {
	var results int

	flag.BoolVar(&debug, "debug", false, "Show debug output")
	flag.BoolVar(&nocolor, "no-color", false, "Uncolorize output")
	flag.BoolVar(&links, "links", false, "Show blockers")
	flag.BoolVar(&markup, "markup", false, "Use markup in output")
	flag.BoolVar(&active, "active", false, "Show only active/open issues in the list")
	flag.IntVar(&results, "results", 50, "Max amount of results per query. Def.: 50")

	flag.Parse()

	args := flag.Args()

	if markup {
		nocolor = true
	}

	color.NoColor = nocolor

	if len(args) != 1 {
		fmt.Println("Usage: jira [-debug] [-no-color] [-results N] [-markup] 'jira query string'")
		os.Exit(0)
	}

	// FIND RIGHT CONFIG FILE TO READ ************************ //
	usr, err := user.Current(); if err != nil {
		panic(err)
	}

	filename := fmt.Sprintf("%s/.jira.json", usr.HomeDir);
	if _, err := os.Stat(filename); err != nil {
		fmt.Printf("There is no config file: %s\n", filename)
		os.Exit(25)
	}

	// ******************************************************* //

	// READ CONFIG FROM KNOWN FILE *************************** //
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		panic(err)
	}

	// ******************************************************* //
	tp := jira.BasicAuthTransport{
		Username: config.User,
		Password: config.Pass,
	}

	jiraClient, _ := jira.NewClient(tp.Client(), config.Host)
	jql := args[0]

	var issues []jira.Issue

	// appendFunc will append jira issues to []jira.Issue
	appendFunc := func(i jira.Issue) (err error) {
		issues = append(issues, i)
		return err
	}

	opts := jira.SearchOptions {
		MaxResults: results }

	jerr := jiraClient.Issue.SearchPages(jql, &opts, appendFunc)
	if jerr != nil {
		fmt.Println(jerr)
		panic(jerr)
	}

	/* userStats
	   - [0] New
	   - [1] Work
	   - [2] Testing
	   - [3] Done
	   - [4] Closed
	   - [5] Story Points
	   - [6] Active Story Points
	*/
	userStats["Total"] = []int{0,0,0,0,0,0,0}
	for _, issue := range issues {
		var linksIsBlockedBy []*jira.IssueLink
		var linksBlocks []*jira.IssueLink

		allIssues[issue.Key] = map[string]bool{}

		// Подсчитать статистику по исполнителям и по статусам
		_, ok := userStats[issue.Fields.Assignee.Name]; if !ok {
			userStats[issue.Fields.Assignee.Name] = []int{0,0,0,0,0,0,0}
		}

		_, ok = statusMap[issue.Fields.Status.Name]; if ok {
			// Do not take into account unknown statuses
			userStats["Total"][statusMap[issue.Fields.Status.Name]] += 1
			userStats[issue.Fields.Assignee.Name][ statusMap[issue.Fields.Status.Name] ] += 1
		}


		// Calculate Story Points by developers
		story_points, has_point := issue.Fields.Unknowns["customfield_10006"]; if has_point {
			if debug { spew.Dump(story_points) }

			points, err := strconv.Atoi( fmt.Sprintf("%.0f", story_points) ); if err == nil {
				userStats["Total"][5] += points
				userStats[issue.Fields.Assignee.Name][5] += points

				if issue.Fields.Status.Name == "Новая" || issue.Fields.Status.Name == "В Работе" {
					userStats["Total"][6] += points
					userStats[issue.Fields.Assignee.Name][6] += points
				}
			}
		}

		for _, link := range issue.Fields.IssueLinks {
			if debug {
				fmt.Println("link.InwardIssue:")
				spew.Dump(link.InwardIssue)

				fmt.Println("link.OutwardIssue:")
				spew.Dump(link.OutwardIssue)
			}

			if link.InwardIssue != nil && link.Type.Inward == "is blocked by" {
				// If blocker is in active status then add it
				_, ok := activeStatus[link.InwardIssue.Fields.Status.Name]; if ok {
					allIssues[issue.Key][link.InwardIssue.Key] = true
				}
				_, ok = activeStatus[issue.Fields.Status.Name]; if ok {
					linksIsBlockedBy = append(linksIsBlockedBy, link)
				}
			}

			if !markup {
				if link.OutwardIssue != nil && link.Type.Outward == "blocks" {
					linksBlocks = append(linksBlocks, link)
				}
			}
		}

		dumpTaskStatus(issue, linksIsBlockedBy, linksBlocks)
		if debug { spew.Dump(issue) }
	}

	fmt.Println()

	if markup {
		fmt.Printf("%s/%s/%s/%s/%s\n", smiles["New"], smiles["Work"], smiles["Test"], smiles["Done"], smiles["Closed"])
	} else {
		fmt.Printf(" N /  W /  T /  D /  C\n")
	}

	dumpUserStat("Total", userStats["Total"])
	delete(userStats, "Total")

	for user, stat := range userStats {
		dumpUserStat(user, stat)
	}


	// Check for blockers outside current list
	fmt.Println()
	fmt.Println("Список блокеров, находящихся вне выборки:")
	for issue, blocks := range allIssues {
		for block, _ := range blocks {
			_, ok := allIssues[block]; if !ok {
				if markup {
					fmt.Printf("%s [%s](%s/browse/%s) заблокирована задачей [%s](%s/browse/%s)\n", smiles["Damn"], issue, config.Host, issue, block, config.Host, block)
				} else {
					fmt.Printf("Issue %-9s is blocked by %-9s which is not in current scope!\n", issue, block)
				}
			}
		}
	}
}

func dumpUserStat(user string, stat []int) {
	if markup {
		fmt.Printf("`%s/%s/%s/%s/%s [%2d/%2d] %s`\n",
			fmt.Sprintf("%2d", stat[0]),
			fmt.Sprintf("%2d", stat[1]),
			fmt.Sprintf("%2d", stat[2]),
			fmt.Sprintf("%2d", stat[3]),
			fmt.Sprintf("%2d", stat[4]),
			stat[6],
			stat[5],
			user)
	} else {
		fmt.Printf("%s / %s / %s / %s / %s\t[%2d / %2d] %s\n",
			statusColourMap["Новая"].Sprintf("%2d", stat[0]),
			statusColourMap["В Работе"].Sprintf("%2d", stat[1]),
			statusColourMap["Тестирование"].Sprintf("%2d", stat[2]),
			statusColourMap["Сделана"].Sprintf("%2d", stat[3]),
			statusColourMap["Закрыта"].Sprintf("%2d", stat[4]),
			stat[6],
			stat[5],
			user)
	}
}

func dumpTaskStatus(issue jira.Issue, blockedBy []*jira.IssueLink, blocks []*jira.IssueLink) {
	var points float64

	_, ok := activeStatus[issue.Fields.Status.Name]; if active && !ok {
		return;
	}

	switch issue.Fields.Unknowns["customfield_10006"].(type) {
		case float64:
			points = issue.Fields.Unknowns["customfield_10006"].(float64)
		default:
			points = 0
	}

	f, ok := statusColourMap[issue.Fields.Status.Name]; if ok && !nocolor {
		f.Printf("%-9s [%2.0f][%-12s] %-30s", issue.Key, points, issue.Fields.Status.Name, issue.Fields.Assignee.Name)

	} else if markup {
		fmt.Printf("[%s](%s/browse/%s) %s [[%3.0f]] *%s* %s",
			issue.Key, config.Host, issue.Key,
			smiles[issue.Fields.Priority.Name],
			points, issue.Fields.Assignee.Name,
			smiles[statusAlias[issue.Fields.Status.Name]])

	} else {
		fmt.Printf("%-9s [%2.0f][%-12s] %-30s", issue.Key, points, issue.Fields.Status.Name, issue.Fields.Assignee.Name)

	}

	if links {
		for _, link := range blockedBy {
			f, ok := statusColourMap[link.InwardIssue.Fields.Status.Name]; if ok && !nocolor {
				f.Printf(" << %s %s (%s);", link.Type.Inward, link.InwardIssue.Key, link.InwardIssue.Fields.Status.Name)

			} else if markup {
				switch statusAlias[link.InwardIssue.Fields.Status.Name] {
					case "New", "Work", "Test":
						fmt.Printf("\n    %s%s%s [%s](%s/browse/%s) %s",
							smiles["Block"], smiles["Block"], smiles["Block"],
							link.InwardIssue.Key, config.Host, link.InwardIssue.Key,
							link.InwardIssue.Fields.Status.Name)
				}

			} else {
				fmt.Printf(" << %s %s (%s);", link.Type.Inward, link.InwardIssue.Key, link.InwardIssue.Fields.Status.Name)

			}
		}

		for _, link := range blocks {
			fmt.Printf(" >> %s %s (%s);", link.Type.Outward, link.OutwardIssue.Key, link.OutwardIssue.Fields.Status.Name)
		}
	}

	fmt.Println()
}
