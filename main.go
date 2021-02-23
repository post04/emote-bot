package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	emutil "github.com/postrequest69/discordgo-emoji-util"
)

// Token I wish the linter would just shut the fuck up
var (
	Prefix    string
	EmojiHelp = &discordgo.MessageEmbed{
		Description: fmt.Sprintf("__Options:__\nadd, addm, delete, edit, list, help, init\n\n__Examples:__\n%vemoji init -- initiates an emoji embed!\n%vemoji add pog https://cdn.discordapp.com/emojis/735224366450606090.png?v=1\n%vemoji delete pog\n%vemoji edit pog pog1\n%vemoji addm :emoji1: :emoji2: :emoji3:....", Prefix, Prefix, Prefix, Prefix, Prefix),
	}
	Database       = make(map[string]*dbStuff)
	Config         config
	GuildPrefixing bool
)

type dbStuff struct {
	Messages  []string `json:"messages"`
	ChannelID string   `json:"channelID"`
}

type config struct {
	Prefix         string `json:"prefix"`
	Token          string `json:"token"`
	GuildPrefixing bool   `json:"guildPrefixing"`
}

func chunks(arr []*discordgo.Emoji) []string {
	var item string
	var toReturn []string
	for _, emoji := range arr {
		if len(item) > 1800 {
			toReturn = append(toReturn, item)
			item = ""
		}
		item += fmt.Sprintf("%v -> `%v`\n", emoji.MessageFormat(), emoji.Name)
	}
	toReturn = append(toReturn, item)
	return toReturn
}

func messageCreate(session *discordgo.Session, msg *discordgo.MessageCreate) {
	if msg.Content == "" {
		return
	}
	parts := strings.Split(msg.Content, " ")
	if len(parts) < 1 {
		return
	}
	command := parts[0]
	if !strings.HasPrefix(command, Prefix) {
		return
	}
	command = strings.ReplaceAll(command, Prefix, "")
	command = strings.ToLower(command)
	parts = parts[1:]
	if command == "em" || command == "emoji" {
		if len(parts) < 1 {
			session.ChannelMessageSendEmbed(msg.ChannelID, EmojiHelp)
			return
		}
		option := strings.ToLower(parts[0])
		switch option {
		case "add":
			// add an emoji
			parts = parts[1:]
			if len(parts) < 2 {
				session.ChannelMessageSendEmbed(msg.ChannelID, EmojiHelp)
				break
			}
			var guildPrefix string
			if GuildPrefixing {
				guild, err := session.State.Guild(msg.GuildID)

				if err == nil {
					for _, g := range strings.Split(guild.Name, " ") {
						guildPrefix += strings.Split(g, "")[0]
					}
					guildPrefix += "_"
				}
			}
			emoji, err := session.GuildEmojiCreate(msg.GuildID, guildPrefix+parts[0], emutil.EncodeImageEmoji(parts[1]), nil)
			if err != nil {
				session.ChannelMessageSend(msg.ChannelID, fmt.Sprintf("```%v```", err))
				break
			}
			session.ChannelMessageSend(msg.ChannelID, fmt.Sprintf("Emoji added %v -> %v", parts[0], emoji.MessageFormat()))
			break
		case "delete":
			// delete an emoji
			parts = parts[1:]
			if len(parts) < 1 {
				session.ChannelMessageSendEmbed(msg.ChannelID, EmojiHelp)
				break
			}
			emojis, _ := session.GuildEmojis(msg.GuildID)
			em := emutil.FindEmoji(emojis, parts[0], false)
			if em == nil {
				session.ChannelMessageSend(msg.ChannelID, "Couldn't find that emoji! Please make sure it's the full name (not case sensitive)")
				break
			}
			err := session.GuildEmojiDelete(msg.GuildID, em.ID)
			if err != nil {
				session.ChannelMessageSend(msg.ChannelID, fmt.Sprintf("```%v```", err))
				break
			}
			session.ChannelMessageSend(msg.ChannelID, "Emoji deleted!")
			break
		case "edit":
			// edit an emoji
			parts = parts[1:]
			if len(parts) < 2 {
				session.ChannelMessageSendEmbed(msg.ChannelID, EmojiHelp)
				break
			}
			emojis, _ := session.GuildEmojis(msg.GuildID)
			em := emutil.FindEmoji(emojis, parts[0], false)
			if em == nil {
				session.ChannelMessageSend(msg.ChannelID, "Couldn't find that emoji! Please make sure it's the full name (not case sensitive)")
				break
			}
			em1, err := session.GuildEmojiEdit(msg.GuildID, em.ID, parts[1], nil)
			if err != nil {
				session.ChannelMessageSend(msg.ChannelID, fmt.Sprintf("```%v```", err))
				break
			}
			session.ChannelMessageSend(msg.ChannelID, fmt.Sprintf("Emoji %v edited to %v", em.MessageFormat(), em1.MessageFormat()))
			break
		case "help":
			// help command
			session.ChannelMessageSendEmbed(msg.ChannelID, EmojiHelp)
			break
		case "init":
			// initiate addition to database
			emojis, _ := session.GuildEmojis(msg.GuildID)
			ems := chunks(emojis)
			//fmt.Println(ems)
			updateEmojis(ems, msg.GuildID, msg.ChannelID, session)
			break
		case "addm":
			emojis := emutil.MatchEmojis(msg.Content)
			if len(emojis) < 1 {
				session.ChannelMessageSend(msg.ChannelID, "No emojis detected!")
				return
			}
			var guildPrefix string
			if GuildPrefixing {
				guild, err := session.State.Guild(msg.GuildID)

				if err == nil {
					for _, g := range strings.Split(guild.Name, " ") {
						guildPrefix += strings.Split(g, "")[0]
					}
					guildPrefix += "_"
				}
			}
			for _, em := range emojis {
				encoded := emutil.EncodeEmojiByID(em.ID)
				if encoded != "" {
					e, err := session.GuildEmojiCreate(msg.GuildID, guildPrefix+em.Name, encoded, nil)
					if err != nil {
						session.ChannelMessageSend(msg.ChannelID, "failed to add "+em.Name)
					}
					session.ChannelMessageSend(msg.ChannelID, em.Name+" -> "+e.MessageFormat())
				} else {
					fmt.Println("test")
				}
			}
		default:
			session.ChannelMessageSendEmbed(msg.ChannelID, EmojiHelp)
			break
		}
	}
}

func every60(session *discordgo.Session) {
	for guild, guilddata := range Database {
		if len(guilddata.Messages) < 1 || guilddata.ChannelID == "" {
			return
		}
		emojis, _ := session.GuildEmojis(guild)
		ems := chunks(emojis)
		updateEmojis(ems, guild, guilddata.ChannelID, session)
	}
}

func sendMessages(channelid string, messages []string, session *discordgo.Session) []string {
	var toReturn []string
	for _, content := range messages {
		msg, err := session.ChannelMessageSend(channelid, content)
		if err == nil {
			toReturn = append(toReturn, msg.ID)
		}

	}
	return toReturn
}

func updateEmojis(messages []string, guildid, channelid string, session *discordgo.Session) {
	if _, ok := Database[guildid]; !ok {
		Database[guildid] = &dbStuff{
			Messages:  []string{},
			ChannelID: channelid,
		}
		saveDatabase()
	}

	if len(Database[guildid].Messages) < 1 {
		if len(messages) < 1 {
			fmt.Println(messages)
			session.ChannelMessageSend(channelid, "This server has no emojis?!")
			return
		}
		Database[guildid].Messages = sendMessages(channelid, messages, session)
		saveDatabase()
		return
	}
	msg, err := session.ChannelMessage(channelid, Database[guildid].Messages[len(Database[guildid].Messages)-1])
	if err != nil {
		_ = session.ChannelMessagesBulkDelete(channelid, Database[guildid].Messages)
		Database[guildid].Messages = sendMessages(channelid, messages, session)
		saveDatabase()
		return
	}
	if len(messages) != len(Database[guildid].Messages) {
		_ = session.ChannelMessagesBulkDelete(channelid, Database[guildid].Messages)
		Database[guildid].Messages = sendMessages(channelid, messages, session)
		saveDatabase()
		return
	}
	if len(msg.Content) < len(messages[len(messages)-1]) {
		_, _ = session.ChannelMessageEdit(channelid, msg.ID, messages[len(messages)-1])
		return
	}
}

func ready(sesion *discordgo.Session, ready *discordgo.Ready) {
	fmt.Println("Logged in as:", ready.User.Username+"#"+ready.User.Discriminator)
	for {
		every60(sesion)
		time.Sleep(60 * time.Second)
	}
}

func getDatabase() {
	db, _ := ioutil.ReadFile("database.json")
	_ = json.Unmarshal(db, &Database)
}

func getConfig() {
	db, _ := ioutil.ReadFile("config.json")
	_ = json.Unmarshal(db, &Config)
}

func saveDatabase() {
	data, _ := json.MarshalIndent(Database, "", "	")
	_ = ioutil.WriteFile("database.json", data, 0666)
}

func main() {
	getDatabase()
	getConfig()
	Prefix = Config.Prefix
	GuildPrefixing = Config.GuildPrefixing
	bot, err := discordgo.New("Bot " + Config.Token)
	if err != nil {
		log.Fatal("ERROR LOGGING IN ", err)
		return
	}
	bot.AddHandler(ready)
	bot.AddHandler(messageCreate)
	err = bot.Open()
	if err != nil {
		log.Fatal("ERROR OPENING CONNECTION ", err)
		return
	}
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	_ = bot.Close()
}
