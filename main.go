package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var (
	BotToken = flag.String("token", "", "Bot access token")
)

type RollNotation struct {
	Addens []Roller
}

type Roller interface {
	Roll() RollResult
}

type RollResult interface {
	Value() int
	fmt.Stringer
}

type RollsSumResult struct {
	Results []RollResult
}

type DieRollResult int

type AcedDieRollResult struct {
	Rolls []DieRollResult
}

type Constant int

type SameDiceSet struct {
	NumDice  int
	NumSides int
}

func (rr DieRollResult) Value() int     { return int(rr) }
func (rr DieRollResult) String() string { return strconv.Itoa(int(rr)) }

func (rr AcedDieRollResult) Value() int {
	sum := 0
	for _, res := range rr.Rolls {
		sum += res.Value()
	}
	return sum
}

func (rr AcedDieRollResult) String() string {
	var rollReprs []string
	for _, roll := range rr.Rolls {
		rollReprs = append(rollReprs, roll.String())
	}
	reprsJoined := strings.Join(rollReprs, " + ")
	if len(rr.Rolls) < 2 {
		return reprsJoined
	}
	return fmt.Sprintf("[%s]", reprsJoined)
}

func (c Constant) Roll() RollResult {
	return DieRollResult(c)
}

func (rr RollsSumResult) Value() int {
	sum := 0
	for _, r := range rr.Results {
		sum += r.Value()
	}
	return sum
}

func (rr RollsSumResult) String() string {
	if len(rr.Results) == 0 {
		return "{ (internal errro) no die results }"
	}
	if len(rr.Results) == 1 {
		return rr.Results[0].String()
	}

	var reprs []string
	for _, r := range rr.Results {
		reprs = append(reprs, r.String())
	}
	return strings.Join(reprs, " + ")
}

func (dc SameDiceSet) Roll() RollResult {
	var results []RollResult
	for i := 0; i < dc.NumDice; i++ {
		roll := rand.Int()%dc.NumSides + 1
		aceRollResult := AcedDieRollResult{Rolls: []DieRollResult{DieRollResult(roll)}}
		for roll == dc.NumSides {
			roll = rand.Int()%dc.NumSides + 1
			aceRollResult.Rolls = append(aceRollResult.Rolls, DieRollResult(roll))
		}
		results = append(results, aceRollResult)
	}
	return RollsSumResult{Results: results}
}

func (rn RollNotation) Roll() RollResult {
	var results []RollResult
	for _, adden := range rn.Addens {
		results = append(results, adden.Roll())
	}
	return RollsSumResult{Results: results}
}

func parseNotation(notation string) (RollNotation, error) {
	var rn RollNotation
	for _, dieNotation := range strings.Split(notation, "+") {
		dieNotation = strings.TrimSpace(dieNotation)
		parts := strings.Split(dieNotation, "d")
		if len(parts) == 0 || len(parts) > 2 {
			return RollNotation{}, fmt.Errorf("expected an adden in [num_dice]d[num_sides] or [constant] format, got: %q", dieNotation)
		}

		var adden Roller
		if len(parts) == 1 {
			constant, err := strconv.Atoi(parts[0])
			if err != nil {
				return RollNotation{}, fmt.Errorf("expected an integer constant or [num_dice]d{num_sides} notation, got: %q", parts[0])
			}
			adden = Constant(constant)
		}
		if len(parts) == 2 {
			numDice := 1
			if parts[0] != "" {
				var err error
				if numDice, err = strconv.Atoi(parts[0]); err != nil {
					return RollNotation{}, fmt.Errorf("expected num_dice to be a natural number in [num_dice]d{num_sides} notation, got: %q", dieNotation)
				}
			}
			numSides, err := strconv.Atoi(parts[1])
			if err != nil {
				return RollNotation{}, fmt.Errorf("expected num_sides to be a natural number in [num_dice]d{num_sides} notation, got: %q", dieNotation)
			}

			adden = SameDiceSet{NumDice: numDice, NumSides: numSides}
		}
		rn.Addens = append(rn.Addens, adden)
	}
	return rn, nil
}

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

	notation, err := parseNotation(notationOpt)
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
