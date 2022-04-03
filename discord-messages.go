package main

import (
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/go-playground/webhooks/v6/github"
	"strings"
)

func messageForGithubPush(Payload github.PushPayload) (*discordgo.MessageEmbed, error) {

	Embed := discordgo.MessageEmbed{}

	Embed.Author = new(discordgo.MessageEmbedAuthor)
	Embed.Author.Name = Payload.Sender.Login
	Embed.Author.URL = Payload.Sender.URL
	Embed.Author.IconURL = Payload.Sender.AvatarURL

	var commitCounter string
	if len(Payload.Commits) == 1 {
		commitCounter = "1 new commit"
	} else {
		commitCounter = fmt.Sprintf("%d new commits", len(Payload.Commits))
	}

	branchName := strings.ReplaceAll(Payload.Ref, "refs/heads/", "")

	Embed.Title = fmt.Sprintf("[%s] %s", Payload.Repository.Name, commitCounter)
	Embed.Thumbnail = new(discordgo.MessageEmbedThumbnail)
	Embed.Thumbnail.URL = "https://riverdark.net/gitgud/git-logo-white.png"
	Embed.Thumbnail.Height = 48
	Embed.Thumbnail.Width = 48

	BranchField := new(discordgo.MessageEmbedField)
	BranchField.Name = "Branch"
	BranchField.Value = branchName
	BranchField.Inline = false

	CommitField := new(discordgo.MessageEmbedField)
	CommitField.Name = "Commit"
	CommitField.Inline = true

	DescField := new(discordgo.MessageEmbedField)
	DescField.Name = "Summary"
	DescField.Inline = true

	for _, commit := range Payload.Commits {
		shortHash := commit.ID[0:6]
		if len(CommitField.Value) > 0 {
			CommitField.Value = CommitField.Value + "\n"
			DescField.Value = DescField.Value + "\n"
		}
		CommitField.Value = CommitField.Value + fmt.Sprintf("[`%s`](%s)", shortHash, commit.URL)

		commitDescription := strings.Split(commit.Message, "\n")[0]
		if len(commitDescription) > 50 {
			commitDescription = commitDescription[0:50]
		}

		DescField.Value = DescField.Value + commitDescription
	}

	Embed.Fields = append(Embed.Fields, BranchField, CommitField, DescField)

	return &Embed, nil
}

func messageForGithubIssue(Payload github.IssuesPayload) (*discordgo.MessageEmbed, error) {

	Embed := discordgo.MessageEmbed{}

	Embed.Author = new(discordgo.MessageEmbedAuthor)
	Embed.Author.Name = Payload.Sender.Login
	Embed.Author.URL = Payload.Sender.URL
	Embed.Author.IconURL = Payload.Sender.AvatarURL

	Embed.Thumbnail = new(discordgo.MessageEmbedThumbnail)
	Embed.Thumbnail.Width = 48
	Embed.Thumbnail.Height = 48

	Embed.Title = fmt.Sprintf("[%s] #%d: %s", Payload.Repository.Name, Payload.Issue.Number, Payload.Issue.Title)
	Embed.URL = Payload.Issue.URL

	StateField := new(discordgo.MessageEmbedField)
	StateField.Name = "Status"
	StateField.Value = Payload.Issue.State
	StateField.Inline = true
	Embed.Fields = append(Embed.Fields, StateField)

	if len(Payload.Issue.Labels) > 0 {
		TagsField := new(discordgo.MessageEmbedField)
		TagsField.Name = "Labels"
		TagsField.Inline = true

		var labels []string
		for _, label := range Payload.Issue.Labels {
			labels = append(labels, label.Name)
		}
		TagsField.Value = strings.Join(labels, ", ")
		Embed.Fields = append(Embed.Fields, TagsField)
	}

	hasAssignee := true

	AssigneeField := new(discordgo.MessageEmbedField)
	AssigneeField.Name = "Assignee"
	AssigneeField.Inline = true
	if Payload.Assignee == nil {
		AssigneeField.Value = "< unassigned >"
		hasAssignee = false
	} else {
		AssigneeField.Value = Payload.Assignee.Login
	}

	switch Payload.Action {
		case "opened":
			Embed.Thumbnail.URL = "https://riverdark.net/gitgud/clipboard-alert.png"

			BugBodyField := new(discordgo.MessageEmbedField)
			BugBodyField.Name = "New Issue"
			BugBodyField.Value = Payload.Issue.Body
			BugBodyField.Inline = false

			Embed.Fields = append(Embed.Fields, BugBodyField)
			break

		case "closed":
			Embed.Thumbnail.URL = "https://riverdark.net/gitgud/clipboard-check.png"
			Embed.Description = "Issue has been closed."
			break

		case "assigned", "unassigned":
			Embed.Thumbnail.URL = "https://riverdark.net/gitgud/clipboard-account.png"
			if Payload.Action == "unassigned" {
				Embed.Description = "Issue has been unassigned."
				hasAssignee = false
			} else {
				Embed.Description = fmt.Sprintf("Issue has been assigned to **[%s](%s)**.", Payload.Assignee.Login, Payload.Assignee.URL)
			}
			break

		default:
			fmt.Println("Got an unfamiliar issue action: ", Payload.Action)
			return nil, errors.New("not an issue change we care about")
	}

	if hasAssignee {
		Embed.Fields = append(Embed.Fields, AssigneeField)
	}

	if Payload.Issue.Milestone != nil {
		MilestoneField := new(discordgo.MessageEmbedField)
		MilestoneField.Name = "Milestone"
		MilestoneField.Value = Payload.Issue.Milestone.Title
		MilestoneField.Inline = true
		Embed.Fields = append(Embed.Fields, MilestoneField)
	}

	return &Embed, nil
}