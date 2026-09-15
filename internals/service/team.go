package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/Anshul1310/tf-register/internals/models"
	"github.com/Anshul1310/tf-register/internals/repository"
	"github.com/google/uuid"
)

type TeamService struct {
	teamRepository *repository.TeamRepository
	userRepository *repository.UserRepository
	maxTeamMembers int
}

func NewTeamService(
	teamRepository *repository.TeamRepository,
	userRepository *repository.UserRepository,
	maxTeamMembers int,
) *TeamService {
	if maxTeamMembers <= 0 {
		maxTeamMembers = 5
	}
	return &TeamService{
		teamRepository: teamRepository,
		userRepository: userRepository,
		maxTeamMembers: maxTeamMembers,
	}
}

func (teamService *TeamService) generateUniqueTeamIdentifier(requestContext context.Context) (string, error) {
	for attemptIndex := 0; attemptIndex < 10; attemptIndex++ {
		randomNumber, generationError := rand.Int(rand.Reader, big.NewInt(900000))
		if generationError != nil {
			return "", fmt.Errorf("failed to generate random team id: %w", generationError)
		}

		candidateTeamIdentifier := fmt.Sprintf("%06d", randomNumber.Int64()+100000)
		existingTeamRecord, queryError := teamService.teamRepository.FindTeamByID(requestContext, candidateTeamIdentifier)
		if queryError != nil {
			return "", fmt.Errorf("failed to check team id collision: %w", queryError)
		}

		if existingTeamRecord == nil {
			return candidateTeamIdentifier, nil
		}
	}

	return "", errors.New("unable to generate unique team id after multiple attempts")
}

func (teamService *TeamService) CreateTeam(
	requestContext context.Context,
	leaderUserID uuid.UUID,
	leaderEmail string,
	createTeamRequest models.CreateTeamRequest,
) (*models.Team, error) {
	trimmedTeamName := strings.TrimSpace(createTeamRequest.Name)
	if trimmedTeamName == "" {
		return nil, errors.New("team name cannot be empty")
	}

	userRecord, findUserError := teamService.userRepository.FindUserByID(requestContext, leaderUserID)
	if findUserError != nil {
		return nil, fmt.Errorf("failed to verify user: %w", findUserError)
	}
	if userRecord == nil {
		return nil, errors.New("user not found; please sync your profile first")
	}
	if userRecord.TeamID != nil && *userRecord.TeamID != "" {
		return nil, errors.New("you are already a member of a team")
	}

	existingTeamWithName, findNameError := teamService.teamRepository.FindTeamByName(requestContext, trimmedTeamName)
	if findNameError != nil {
		return nil, fmt.Errorf("failed to check existing team name: %w", findNameError)
	}
	if existingTeamWithName != nil {
		return nil, errors.New("a team already exists with that name. Please choose another name")
	}

	uniqueTeamIdentifier, generateError := teamService.generateUniqueTeamIdentifier(requestContext)
	if generateError != nil {
		return nil, fmt.Errorf("failed to generate team identifier: %w", generateError)
	}

	leaderContactInformation := strings.TrimSpace(createTeamRequest.Contact)
	if leaderContactInformation == "" {
		leaderContactInformation = "N/A"
	}

	effectiveLeaderEmail := leaderEmail
	if effectiveLeaderEmail == "" {
		effectiveLeaderEmail = userRecord.Email
	}

	teamToCreate := &models.Team{
		TeamID:           uniqueTeamIdentifier,
		Name:             trimmedTeamName,
		Leader:           effectiveLeaderEmail,
		LeaderUserID:     leaderUserID,
		Contact:          leaderContactInformation,
		Domain:           createTeamRequest.Domain,
		ProblemStatement: createTeamRequest.ProblemStatement,
		PaymentStatus:    "Pending",
		IsPublic:         false,
	}

	creationError := teamService.teamRepository.CreateTeamWithLeader(requestContext, teamToCreate)
	if creationError != nil {
		return nil, fmt.Errorf("failed to create team: %w", creationError)
	}

	return teamToCreate, nil
}

func (teamService *TeamService) JoinTeam(
	requestContext context.Context,
	joiningUserID uuid.UUID,
	teamIdentifier string,
) error {
	trimmedTeamIdentifier := strings.TrimSpace(teamIdentifier)
	if trimmedTeamIdentifier == "" {
		return errors.New("team code is required")
	}

	userRecord, findUserError := teamService.userRepository.FindUserByID(requestContext, joiningUserID)
	if findUserError != nil {
		return fmt.Errorf("failed to check user profile: %w", findUserError)
	}
	if userRecord == nil {
		return errors.New("user not found")
	}
	if userRecord.TeamID != nil && *userRecord.TeamID != "" {
		return errors.New("you are already part of a team")
	}

	teamRecord, findTeamError := teamService.teamRepository.FindTeamByID(requestContext, trimmedTeamIdentifier)
	if findTeamError != nil {
		return fmt.Errorf("failed to verify team: %w", findTeamError)
	}
	if teamRecord == nil {
		return errors.New("team with the specified code was not found")
	}

	currentMemberCount, countError := teamService.teamRepository.CountTeamMembers(requestContext, trimmedTeamIdentifier)
	if countError != nil {
		return fmt.Errorf("failed to count team members: %w", countError)
	}

	if currentMemberCount >= teamService.maxTeamMembers {
		return fmt.Errorf("maximum capacity reached! Team already has %d members", currentMemberCount)
	}

	updateError := teamService.userRepository.UpdateUserTeam(requestContext, joiningUserID, &trimmedTeamIdentifier)
	if updateError != nil {
		return fmt.Errorf("failed to add user to team: %w", updateError)
	}

	return nil
}

func (teamService *TeamService) GetTeamDetails(requestContext context.Context, teamIdentifier string) (*models.Team, error) {
	teamRecord, findTeamError := teamService.teamRepository.FindTeamByIDWithMembers(requestContext, teamIdentifier)
	if findTeamError != nil {
		return nil, fmt.Errorf("failed to retrieve team: %w", findTeamError)
	}
	if teamRecord == nil {
		return nil, errors.New("team not found")
	}

	return teamRecord, nil
}

func (teamService *TeamService) GetPublicTeams(
	requestContext context.Context,
	domainFilter string,
	searchQuery string,
) ([]models.Team, error) {
	publicTeamsList, queryError := teamService.teamRepository.FindPublicTeams(requestContext, domainFilter, searchQuery)
	if queryError != nil {
		return nil, fmt.Errorf("failed to retrieve public teams: %w", queryError)
	}

	return publicTeamsList, nil
}

func (teamService *TeamService) GetTeamMemberCount(requestContext context.Context, teamIdentifier string) (int, error) {
	memberCount, queryError := teamService.teamRepository.CountTeamMembers(requestContext, teamIdentifier)
	if queryError != nil {
		return 0, fmt.Errorf("failed to retrieve member count: %w", queryError)
	}

	return memberCount, nil
}

func (teamService *TeamService) GetPaidTeamCount(requestContext context.Context) (int, error) {
	paidTeamCount, queryError := teamService.teamRepository.CountPaidTeams(requestContext)
	if queryError != nil {
		return 0, fmt.Errorf("failed to retrieve paid team count: %w", queryError)
	}

	return paidTeamCount, nil
}

func (teamService *TeamService) VerifyTeamLeader(requestContext context.Context, requesterUserID uuid.UUID, teamIdentifier string) (*models.Team, error) {
	teamRecord, findTeamError := teamService.teamRepository.FindTeamByID(requestContext, teamIdentifier)
	if findTeamError != nil {
		return nil, fmt.Errorf("failed to find team: %w", findTeamError)
	}
	if teamRecord == nil {
		return nil, errors.New("team not found")
	}

	if teamRecord.LeaderUserID != requesterUserID {
		return nil, errors.New("only the team leader is permitted to perform this action")
	}

	return teamRecord, nil
}

func (teamService *TeamService) UpdateVisibility(
	requestContext context.Context,
	requesterUserID uuid.UUID,
	teamIdentifier string,
	isPublic bool,
) error {
	_, authorizationError := teamService.VerifyTeamLeader(requestContext, requesterUserID, teamIdentifier)
	if authorizationError != nil {
		return authorizationError
	}

	updateError := teamService.teamRepository.UpdateTeamVisibility(requestContext, teamIdentifier, isPublic)
	if updateError != nil {
		return fmt.Errorf("failed to update visibility: %w", updateError)
	}

	return nil
}

func (teamService *TeamService) UpdateTeamName(
	requestContext context.Context,
	requesterUserID uuid.UUID,
	teamIdentifier string,
	newTeamName string,
) error {
	_, authorizationError := teamService.VerifyTeamLeader(requestContext, requesterUserID, teamIdentifier)
	if authorizationError != nil {
		return authorizationError
	}

	trimmedName := strings.TrimSpace(newTeamName)
	if trimmedName == "" {
		return errors.New("new team name cannot be empty")
	}

	existingTeam, findNameError := teamService.teamRepository.FindTeamByName(requestContext, trimmedName)
	if findNameError != nil {
		return fmt.Errorf("failed to check team name: %w", findNameError)
	}
	if existingTeam != nil && existingTeam.TeamID != teamIdentifier {
		return errors.New("a team already exists with that name")
	}

	updateError := teamService.teamRepository.UpdateTeamName(requestContext, teamIdentifier, trimmedName)
	if updateError != nil {
		return fmt.Errorf("failed to update team name: %w", updateError)
	}

	return nil
}

func (teamService *TeamService) UpdateTeamDetails(
	requestContext context.Context,
	requesterUserID uuid.UUID,
	teamIdentifier string,
	domain *string,
	problemStatement *string,
) error {
	teamRecord, authorizationError := teamService.VerifyTeamLeader(requestContext, requesterUserID, teamIdentifier)
	if authorizationError != nil {
		return authorizationError
	}

	if teamRecord.PaymentStatus == "PAID" && domain != nil && *domain != "" {
		if teamRecord.Domain != nil && *teamRecord.Domain != *domain {
			return errors.New("domain cannot be changed after payment has been completed")
		}
	}

	updateError := teamService.teamRepository.UpdateTeamDetails(requestContext, teamIdentifier, domain, problemStatement)
	if updateError != nil {
		return fmt.Errorf("failed to update team details: %w", updateError)
	}

	return nil
}

func (teamService *TeamService) RegenerateTeamID(
	requestContext context.Context,
	requesterUserID uuid.UUID,
	currentTeamID string,
) (string, error) {
	_, authorizationError := teamService.VerifyTeamLeader(requestContext, requesterUserID, currentTeamID)
	if authorizationError != nil {
		return "", authorizationError
	}

	newUniqueTeamID, generateError := teamService.generateUniqueTeamIdentifier(requestContext)
	if generateError != nil {
		return "", fmt.Errorf("failed to generate new team id: %w", generateError)
	}

	updateError := teamService.teamRepository.UpdateTeamID(requestContext, currentTeamID, newUniqueTeamID)
	if updateError != nil {
		return "", fmt.Errorf("failed to update team id: %w", updateError)
	}

	return newUniqueTeamID, nil
}

func (teamService *TeamService) DeleteTeam(
	requestContext context.Context,
	requesterUserID uuid.UUID,
	teamIdentifier string,
) error {
	_, authorizationError := teamService.VerifyTeamLeader(requestContext, requesterUserID, teamIdentifier)
	if authorizationError != nil {
		return authorizationError
	}

	deleteError := teamService.teamRepository.DeleteTeam(requestContext, teamIdentifier)
	if deleteError != nil {
		return fmt.Errorf("failed to delete team: %w", deleteError)
	}

	return nil
}

func (teamService *TeamService) SubmitManualPayment(
	requestContext context.Context,
	teamIdentifier string,
	transactionID string,
	screenshotURL string,
) error {
	trimmedTransactionID := strings.TrimSpace(transactionID)
	if trimmedTransactionID == "" {
		return errors.New("transaction id is required")
	}

	memberCount, countError := teamService.teamRepository.CountTeamMembers(requestContext, teamIdentifier)
	if countError != nil {
		return fmt.Errorf("failed to count team members: %w", countError)
	}

	if memberCount < 4 {
		return fmt.Errorf("you need to have at least 4 members in your team to submit payment (current: %d)", memberCount)
	}

	updateError := teamService.teamRepository.UpdatePaymentStatus(requestContext, teamIdentifier, "Processing")
	if updateError != nil {
		return fmt.Errorf("failed to update payment status: %w", updateError)
	}

	logPaymentError := teamService.teamRepository.RecordPaymentLog(
		requestContext,
		fmt.Sprintf("manual_%s", trimmedTransactionID),
		fmt.Sprintf("order_%s", teamIdentifier),
		teamIdentifier,
		200.0,
		"Processing",
		"",
		trimmedTransactionID,
		screenshotURL,
		"",
	)
	if logPaymentError != nil {
		fmt.Printf("Warning: failed to record payment audit log: %v\n", logPaymentError)
	}

	return nil
}
