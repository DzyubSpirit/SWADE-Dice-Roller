package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/DzyubSpirit/swade-dice-roller/v2/roll"
	"github.com/bwmarrin/discordgo"
)

var (
	BotToken = flag.String("token", "", "Bot access token")
)

func handler(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	opts := interaction.ApplicationCommandData().Options
	var notationOpt string
	for _, opt := range opts {
		if opt.Name == "notation" {
			notationOpt = opt.StringValue()
		}
	}
	if notationOpt == "" {
		session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "/roll command must have a 'notation' option value",
			},
		})
		return
	}

	notation, err := roll.ParseNotation(notationOpt)
	if err != nil {
		session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("/roll command had an error: %v", err),
			},
		})
		return
	}

	rollResult := notation.Roll()
	rollFormatted := fmt.Sprintf("%s = %v", rollResult, rollResult.Value())
	session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("The roll for %s:\n  %s", notationOpt, rollFormatted),
		},
	})
}

func main() {
	flag.Parse()

	discord, err := discordgo.New("Bot " + *BotToken)
	if err != nil {
		log.Fatalf("discordgo.New(): %v", err)
	}

	if err := discord.Open(); err != nil {
		log.Fatalf("Open Discord socket, error: %v", err)
	}
	defer discord.Close()

	discord.AddHandler(handler)
	_, err = discord.ApplicationCommandCreate(discord.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "roll",
		Description: "Roll a dice for a trait roll in Savage Worlds. The dice ace and the best of roll is chosen",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "notation",
				Description: "Dice roll notation for the roll to make",
				Required:    true,
			},
		},
	})
	if err != nil {
		log.Fatalf("Create roll command error: %v", err)
	}

	println("Bot initalized")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-stop
}
