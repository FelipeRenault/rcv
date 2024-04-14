package main

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	path = "D:/Users/Felipe/Documents/Go/rcv/test.csv"
)

var (
	categoryTitleRegex     = regexp.MustCompile(`(.*) \[`)
	categoryCandidateRegex = regexp.MustCompile(`\[(.*)\]`)
)

type Ballot map[string][]string

type Category struct {
	Title      string
	Candidates []string
	Ballots    [][]string
}

type Categories []Category

func (category Category) PrettyPrint() {
	fmt.Println("Categoria:", category.Title)
	fmt.Println("\t↳ Candidatos:", strings.Join(category.Candidates, ", "))
	fmt.Println("\t↳ Cédulas:")
	for _, ballot := range category.Ballots {
		fmt.Println("\t\t↳", strings.Join(ballot, ", "))
	}
}

func main() {
	content, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	lines := strings.Split(string(content), "\n")
	categories := parseCategories(lines[0])
	parseBallots(categories, lines[1:])
	for _, category := range categories {
		injectCandidates(&category)
		category.PrettyPrint()
		winners, honorableMention := determineWinner(category, CandidateVotes{})
		if winners[0].Votes[0] > winners[1].Votes[0] {
			fmt.Printf("\nGanhador determinado para a categoria %s!\n", category.Title)
			fmt.Printf("\t↳ %s (%d votos)\n", winners[0].Candidate, winners[0].Votes[0])
			fmt.Printf("Segundo lugar: %s (%d votos)\n", winners[1].Candidate, winners[1].Votes[0])
			if honorableMention.Candidate != winners[0].Candidate {
				fmt.Printf("Menção honrosa: %s (%d votos)\n", honorableMention.Candidate, honorableMention.Votes[0])
			}
		} else {
			fmt.Printf("\nEmpate na categoria %s!\n", category.Title)
			fmt.Printf("\t↳ %s | %s (%d votos)\n", winners[0].Candidate, winners[1].Candidate, winners[0].Votes[0])
			if honorableMention.Candidate != winners[0].Candidate && honorableMention.Candidate != winners[1].Candidate {
				fmt.Printf("Menção honrosa: %s (%d votos)\n", honorableMention.Candidate, honorableMention.Votes[0])
			}
		}
		fmt.Printf("\n")
	}
}

func parseCategories(headerLine string) Categories {
	headers := strings.Split(headerLine, `","`)
	categories := make(Categories, 0)
	var (
		currTitle    string
		currCategory Category
	)
	for _, voteColumn := range headers[1:] {
		title := voteColumn
		var candidate string
		if titleMatch := categoryTitleRegex.FindStringSubmatch(voteColumn); len(titleMatch) > 1 {
			title = categoryTitleRegex.FindStringSubmatch(voteColumn)[1]
			candidate = categoryCandidateRegex.FindStringSubmatch(voteColumn)[1]
		}
		if title != currTitle {
			if currTitle != "" {
				categories = append(categories, currCategory)
			}
			currCategory = Category{
				Title:      title,
				Candidates: make([]string, 0),
			}
			currTitle = title
		}
		if candidate != "" {
			currCategory.Candidates = append(currCategory.Candidates, candidate)
		}
	}
	categories = append(categories, currCategory)

	return categories
}

func parseBallots(categories Categories, voteLines []string) {
	for _, voteLine := range voteLines {
		votes := strings.Split(voteLine, `","`)
		var (
			categoryPos int
			votePos     int
		)
		ballot := make([]string, len(categories[categoryPos].Candidates))
		for _, vote := range votes[1:] {
			currCategory := &categories[categoryPos]
			vote := strings.TrimSuffix(vote, `"`)
			vote = strings.TrimSuffix(vote, `º`)
			if len(vote) > 0 {
				pos, err := strconv.Atoi(vote)
				if err != nil {
					ballot = append(ballot, vote)
				} else {
					ballot[pos-1] = categories[categoryPos].Candidates[votePos]
				}
			}
			votePos++
			if votePos >= len(currCategory.Candidates) {
				currCategory.Ballots = append(currCategory.Ballots, removeEmptyVotes(ballot))
				categoryPos++
				votePos = 0
				if categoryPos < len(categories) {
					ballot = make([]string, len(categories[categoryPos].Candidates))
				}
			}
		}
	}
	return
}

func removeEmptyVotes(ballot []string) []string {
	for i, candidate := range ballot {
		if candidate == "" {
			return ballot[:i]
		}
	}
	return ballot
}

// Parse candidates from ballots for non-rcv voting
func injectCandidates(category *Category) {
	if len(category.Candidates) == 0 {
		candidates := make(map[string]struct{})
		for _, ballot := range category.Ballots {
			candidates[ballot[0]] = struct{}{}
		}
		for candidate := range candidates {
			category.Candidates = append(category.Candidates, candidate)
		}
	}
}

func determineWinner(category Category, honorableMention CandidateVotes) (CandidatesVotes, CandidateVotes) {
	fmt.Println("Contagem de votos!")
	candidatesVotes, validVotes := votesPerCandidate(category)

	cv := rankCandidates(candidatesVotes)
	cv.PrettyPrint()

	if honorableMention.Candidate == "" {
		honorableMention = cv[0]
	}

	if winners := checkWinners(cv, validVotes); len(winners) > 0 {
		return winners, honorableMention
	}

	last := cv[len(cv)-1].Candidate
	removeLast(&category, last)

	fmt.Println("Recontagem necessária...")

	return determineWinner(category, honorableMention)
}

func votesPerCandidate(category Category) (map[string][]int, int) {
	candidatesVotes := make(map[string][]int, len(category.Candidates))
	for _, candidate := range category.Candidates {
		candidatesVotes[candidate] = make([]int, len(category.Candidates))
	}

	var validVotes int
	for _, ballot := range category.Ballots {
		for i, vote := range ballot {
			candidatesVotes[vote][i]++
			if i == 0 {
				validVotes++
			}
		}
	}

	return candidatesVotes, validVotes
}

type CandidateVotes struct {
	Candidate string
	Votes     []int
}

type CandidatesVotes []CandidateVotes

func (cv CandidatesVotes) PrettyPrint() {
	for rank, candidate := range cv {
		var text []string
		for pos, votes := range candidate.Votes {
			text = append(text, fmt.Sprintf("%dº: %d", pos+1, votes))
		}
		fmt.Printf("\t↳ %dº: %s [%s]\n", rank+1, candidate.Candidate, strings.Join(text, ", "))
	}
}

func rankCandidates(candidateVotes map[string][]int) CandidatesVotes {
	cv := make(CandidatesVotes, len(candidateVotes))
	var i int
	for candidate, votes := range candidateVotes {
		cv[i] = CandidateVotes{
			Candidate: candidate,
			Votes:     votes,
		}
		i++
	}

	sort.Slice(cv, func(i, j int) bool {
		for k := range cv[i].Votes {
			if cv[i].Votes[k] == cv[j].Votes[k] {
				continue
			}
			return cv[i].Votes[k] > cv[j].Votes[k]
		}
		return false
	})

	return cv
}

func checkWinners(cv CandidatesVotes, validVotes int) CandidatesVotes {
	if cv[0].Votes[0] > validVotes/2 || len(cv) == 2 {
		return cv[:2]
	}

	return CandidatesVotes{}
}

func removeLast(category *Category, last string) {
	fmt.Println("Removendo o último colocado:", last)
	for j, ballot := range category.Ballots {
		for i, vote := range ballot {
			if vote == last {
				category.Ballots[j] = append(ballot[:i], ballot[i+1:]...)
			}
		}
	}
	for i, candidate := range category.Candidates {
		if candidate == last {
			category.Candidates = append(category.Candidates[:i], category.Candidates[i+1:]...)
			break
		}
	}
}
