package roll

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

type RollNotation struct {
	Addens []Roller
}

type Roller interface {
	Roll() RollResult
	SetRand(*rand.Rand)
}

type RollResult interface {
	Value() int
	fmt.Stringer
	Detailed(wrapped bool) string
}

type SumRollResult struct {
	Results []RollResult
}

type BestRollResult struct {
	Results []RollResult
}

type DieRollResult int

type AcedDieRollResult struct {
	Rolls []DieRollResult
}

type Constant int

type DiceSets struct {
	NumSets int
	DiceSet []int

	rand *rand.Rand
}

func (rr DieRollResult) Value() int                   { return int(rr) }
func (rr DieRollResult) Detailed(wrapped bool) string { return strconv.Itoa(int(rr)) }
func (rr DieRollResult) String() string               { return rr.Detailed(false) }

func (rr AcedDieRollResult) Value() int {
	sum := 0
	for _, res := range rr.Rolls {
		sum += res.Value()
	}
	return sum
}

func (rr AcedDieRollResult) Detailed(wrapped bool) string {
	if len(rr.Rolls) < 2 {
		log.Printf("AcedDieRollResult.Detailed(%v): rr.Rolls has less than 2 rolls: %q. Want 2 or more rolls", wrapped, rr.Rolls)
		return ""
	}
	var rollReprs []string
	for _, roll := range rr.Rolls {
		rollReprs = append(rollReprs, roll.Detailed( /*wrapped=*/ true))
	}
	rollRepr := strings.Join(rollReprs, " + ")
	if wrapped {
		return "[" + rollRepr + "]"
	} else {
		return rollRepr
	}
}

func (rr AcedDieRollResult) String() string { return rr.Detailed(false) }

func (c Constant) SetRand(*rand.Rand) {}
func (c Constant) Roll() RollResult {
	return DieRollResult(c)
}

func (rr SumRollResult) Value() int {
	sum := 0
	for _, r := range rr.Results {
		sum += r.Value()
	}
	return sum
}

func (rr SumRollResult) Detailed(wrapped bool) string {
	if len(rr.Results) == 0 {
		log.Println("SumRollResult.String() has an empty results list")
		return "{ (internal error) no die results }"
	}
	if len(rr.Results) == 1 {
		return rr.Results[0].Detailed(wrapped)
	}

	var reprs []string
	for _, r := range rr.Results {
		reprs = append(reprs, r.Detailed( /*wrapped=*/ true))
	}
	return strings.Join(reprs, " + ")
}

func (rr SumRollResult) String() string { return rr.Detailed(false) }

func (rr BestRollResult) Detailed(wrapped bool) string {
	if len(rr.Results) == 0 {
		log.Println("BestRollResult.String() has an empty results list")
		return " { (internal error) no die results }"
	}
	if len(rr.Results) == 1 {
		return rr.Results[0].Detailed(wrapped)
	}

	var reprs []string
	for _, r := range rr.Results {
		reprs = append(reprs, r.Detailed( /*wrapped=*/ false))
	}
	return fmt.Sprintf("[%s] %v", strings.Join(reprs, ", "), rr.Value())
}

func (rr BestRollResult) String() string { return rr.Detailed(false) }

func (rr BestRollResult) Value() int {
	if len(rr.Results) == 0 {
		log.Println("BestRollResult.Value() has an empty results list")
		return 1
	}
	maxResult := rr.Results[0].Value()
	for _, result := range rr.Results[1:] {
		if result.Value() > maxResult {
			maxResult = result.Value()
		}
	}
	return maxResult
}

func NewDiceSets(numSets int, diceSet []int) *DiceSets {
	return &DiceSets{
		NumSets: numSets, DiceSet: diceSet,
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (dc *DiceSets) SetRand(r *rand.Rand) {
	dc.rand = r
}

func (dc DiceSets) Roll() RollResult {
	var results []RollResult
	for i := 0; i < dc.NumSets; i++ {
		for _, die := range dc.DiceSet {
			roll := dc.rand.Int()%die + 1
			if roll < die {
				results = append(results, DieRollResult(roll))
				continue
			}

			rollResult := AcedDieRollResult{Rolls: []DieRollResult{DieRollResult(roll)}}
			for roll == die {
				roll = dc.rand.Int()%die + 1
				rollResult.Rolls = append(rollResult.Rolls, DieRollResult(roll))
			}
			results = append(results, rollResult)
		}
	}
	return BestRollResult{Results: results}
}

func (rn RollNotation) Roll() RollResult {
	var results []RollResult
	for _, adden := range rn.Addens {
		results = append(results, adden.Roll())
	}
	return SumRollResult{Results: results}
}

func (rn *RollNotation) SetRand(r *rand.Rand) {
	for _, adden := range rn.Addens {
		adden.SetRand(r)
	}
}

func parseDNotation(dParts []string) (Roller, error) {
	numSetsStr, diceStrs := dParts[0], dParts[1:]
	numSets := 1
	if numSetsStr != "" {
		var err error
		if numSets, err = strconv.Atoi(numSetsStr); err != nil {
			return &RollNotation{}, fmt.Errorf("expected num_dice to be a natural number in [num_dice]d{num_sides} notation, got: %q", numSetsStr)
		}
	}

	var dice []int
	for _, dieStr := range diceStrs {
		die, err := strconv.Atoi(dieStr)
		if err != nil {
			return &RollNotation{}, fmt.Errorf("expected num_sides to be a natural number in [num_dice]d{num_sides} notation, got: %q", dieStr)
		}
		dice = append(dice, die)
	}
	return NewDiceSets(numSets, dice), nil
}

func ParseNotation(notation string) (RollNotation, error) {
	var rn RollNotation
	for _, dieNotation := range strings.Split(notation, "+") {
		dieNotation = strings.TrimSpace(dieNotation)
		parts := strings.Split(dieNotation, "d")

		if len(parts) == 1 {
			constant, err := strconv.Atoi(parts[0])
			if err != nil {
				return RollNotation{}, fmt.Errorf("expected an integer constant or [num_dice]d{num_sides} notation, got: %q", parts[0])
			}
			rn.Addens = append(rn.Addens, Constant(constant))
			continue
		}

		if len(parts) > 1 {
			adden, err := parseDNotation(parts)
			if err != nil {
				return RollNotation{}, fmt.Errorf("failed to parse d notation for %q: %v", dieNotation, err)
			}
			rn.Addens = append(rn.Addens, adden)
		}
	}
	return rn, nil
}
