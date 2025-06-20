package main

import "fmt"

func main() {
	userName := "shirazhaiderkr"
	userEmail := "shirazhaiderkr@gmail.com"
	isPremiumUser := true
	userCredits := 1000

	fmt.Printf("User:%s\n Username: %s\n Are they premium user: %t\n How much credits do they have: %d\n", userName, userEmail, isPremiumUser, userCredits)

	createProject("BlairProject1", "sheerazhaiderkr@gmail.com")
	createProject("BlairProject2", "sheerazhaiderkr2@gmail.com")
	simulateGenAICost(20, userCredits)
	simulateGenAICost(50, userCredits)
	simulateGenAICost(1000, userCredits)

}
func createProject(projectName string, projectOwnerEmail string) {

	fmt.Printf("Project %s created by %s\n", projectName, projectOwnerEmail)

}

func simulateGenAICost(createVariationsCount int, userCredits int) {

	totalCost := 5 * createVariationsCount

	if totalCost > userCredits {
		fmt.Print("Not enough credits\n")

	} else {
		fmt.Printf("GenAI Produced\n")
	}

}
