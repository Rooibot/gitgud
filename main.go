package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/go-playground/webhooks/v6/github"
	"gopkg.in/yaml.v2"
	"net/http"
	"os"
	"strconv"
)

const (
	path = "/webhooks"
)

type GitgudConfig struct {
	Server struct {
		Port int    `yaml:"port"`
		Path string `yaml:"path"`
	} `yaml:"server"`
	Discord struct {
		Secret  string `yaml:"secret"`
		Channel string `yaml:"channel"`
	} `yaml:"discord"`
	Github struct {
		Secret string `yaml:"secret"`
	} `yaml:"github"`
}

func sendDiscordMessage(s *discordgo.Session, channel string, Embed *discordgo.MessageEmbed) (*discordgo.Message, error) {

	return s.ChannelMessageSendEmbed(channel, Embed)
}

func ready(s *discordgo.Session, event *discordgo.Ready) {

	s.UpdateGameStatus(0, "Creepin' on your codebase")

}

func guildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {

}

func readConfigFile() (*GitgudConfig, error) {

	var configFile string

	flag.StringVar(&configFile, "config", "./config.yml", "The path to a valid config file.")
	flag.Parse()

	s, err := os.Stat(configFile)
	if nil != err {
		return nil, err
	}
	if s.IsDir() {
		return nil, errors.New("file given was a directory")
	}

	config := &GitgudConfig{}

	file, fileErr := os.OpenFile(configFile, os.O_RDONLY, 0644)
	if fileErr != nil {
		return nil, fileErr
	}
	defer file.Close()

	yamlDecode := yaml.NewDecoder(file)
	if yamlErr := yamlDecode.Decode(&config); yamlErr != nil {
		return nil, yamlErr
	}

	if len(config.Server.Path) == 0 {
		config.Server.Path = "/webhooks"
	}

	if config.Server.Port == 0 {
		config.Server.Port = 3412
	}

	if len(config.Discord.Secret) == 0 {
		return nil, errors.New("config file was missing a Discord secret")
	}

	if len(config.Discord.Channel) == 0 {
		return nil, errors.New("config file was missing a Discord channel")
	}

	if len(config.Github.Secret) == 0 {
		return nil, errors.New("config file was missing a Github secret")
	}

	return config, nil
}

func main() {

	config, err := readConfigFile()
	if err != nil {
		println(err)
		os.Exit(1)
	}

	discord, err := discordgo.New(config.Discord.Secret)
	if err != nil {
		fmt.Println("error creating Discord session: ", err)
		return
	}

	discord.AddHandler(ready)
	discord.AddHandler(guildCreate)

	discord.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages

	err = discord.Open()
	if err != nil {
		fmt.Println("error opening Discord session: ", err)
	}

	hook, _ := github.New(github.Options.Secret(config.Github.Secret))

	http.HandleFunc(config.Server.Path, func(w http.ResponseWriter, r *http.Request) {
		payload, err := hook.Parse(r, github.PushEvent, github.IssuesEvent)
		if err != nil {
			if err == github.ErrEventNotFound {
				// ok event wasn;t one of the ones asked to be parsed
			}
		}

		switch payload.(type) {

		case github.PushPayload:
			PushMessage, _ := messageForGithubPush(payload.(github.PushPayload))
			if PushMessage != nil {
				fmt.Println("Sent a push summary!")
				sendDiscordMessage(discord, config.Discord.Channel, PushMessage)
			}
			break

		case github.IssuesPayload:
			IssueMessage, _ := messageForGithubIssue(payload.(github.IssuesPayload))
			if IssueMessage != nil {
				fmt.Println("Sent an issue summary!")
				sendDiscordMessage(discord, config.Discord.Channel, IssueMessage)
			}
			break
		}
	})
	http.ListenAndServe(":"+strconv.Itoa(config.Server.Port), nil)

	discord.Close()

}
