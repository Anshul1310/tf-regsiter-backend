package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Anshul1310/tf-register/config"
	"github.com/Anshul1310/tf-register/internals/models"
	"github.com/Anshul1310/tf-register/internals/repository"
	"github.com/google/uuid"
)

type UserService struct {
	userRepository *repository.UserRepository
	teamRepository *repository.TeamRepository
	config         *config.Config
	httpClient     *http.Client
}

func NewUserService(
	userRepository *repository.UserRepository,
	teamRepository *repository.TeamRepository,
	config *config.Config,
) *UserService {
	return &UserService{
		userRepository: userRepository,
		teamRepository: teamRepository,
		config:         config,
		httpClient:     &http.Client{Timeout: 15 * time.Second},
	}
}

func (userService *UserService) GetUserProfile(requestContext context.Context, targetUserID uuid.UUID) (*models.User, error) {
	userRecord, findUserError := userService.userRepository.FindUserByID(requestContext, targetUserID)
	if findUserError != nil {
		return nil, fmt.Errorf("failed to retrieve user profile: %w", findUserError)
	}

	if userRecord == nil {
		return nil, errors.New("user not found")
	}

	return userRecord, nil
}

func (userService *UserService) SyncUser(requestContext context.Context, syncRequest models.SyncUserRequest) (*models.User, error) {
	if syncRequest.UserID == uuid.Nil {
		return nil, errors.New("user id cannot be empty")
	}

	userModelToSave := &models.User{
		UserID: syncRequest.UserID,
		Email:  syncRequest.Email,
		Name:   syncRequest.Name,
		Pfp:    &syncRequest.Pfp,
	}

	savedUserRecord, upsertError := userService.userRepository.UpsertUser(requestContext, userModelToSave)
	if upsertError != nil {
		return nil, fmt.Errorf("failed to sync user: %w", upsertError)
	}

	return savedUserRecord, nil
}

func (userService *UserService) UpdateProfile(
	requestContext context.Context,
	targetUserID uuid.UUID,
	profileUpdateRequest models.UpdateUserProfileRequest,
) (*models.User, error) {
	existingUserRecord, findUserError := userService.userRepository.FindUserByID(requestContext, targetUserID)
	if findUserError != nil {
		return nil, fmt.Errorf("failed to find existing user: %w", findUserError)
	}

	if existingUserRecord == nil {
		return nil, errors.New("user not found")
	}

	userProfileToUpdate := &models.User{
		UserID:     targetUserID,
		Name:       profileUpdateRequest.Name,
		RollNumber: &profileUpdateRequest.RollNumber,
		Hostel:     &profileUpdateRequest.Hostel,
		Mess:       &profileUpdateRequest.Mess,
		Gender:     &profileUpdateRequest.Gender,
	}

	if profileUpdateRequest.Email != nil && *profileUpdateRequest.Email != "" {
		userProfileToUpdate.Email = *profileUpdateRequest.Email
	} else {
		userProfileToUpdate.Email = existingUserRecord.Email
	}

	updatedUserRecord, updateError := userService.userRepository.UpdateUserProfile(requestContext, userProfileToUpdate)
	if updateError != nil {
		return nil, fmt.Errorf("failed to update user profile: %w", updateError)
	}

	return updatedUserRecord, nil
}

func (userService *UserService) LeaveTeam(requestContext context.Context, targetUserID uuid.UUID) error {
	userRecord, findUserError := userService.userRepository.FindUserByID(requestContext, targetUserID)
	if findUserError != nil {
		return fmt.Errorf("failed to find user: %w", findUserError)
	}

	if userRecord == nil {
		return errors.New("user not found")
	}

	if userRecord.TeamID == nil || *userRecord.TeamID == "" {
		return errors.New("user is not currently in any team")
	}

	// Check if user is the leader of the team
	teamRecord, findTeamError := userService.teamRepository.FindTeamByID(requestContext, *userRecord.TeamID)
	if findTeamError == nil && teamRecord != nil {
		if teamRecord.LeaderUserID == targetUserID {
			return errors.New("team leader cannot leave the team; delete the team instead")
		}
	}

	updateError := userService.userRepository.UpdateUserTeam(requestContext, targetUserID, nil)
	if updateError != nil {
		return fmt.Errorf("failed to remove user from team: %w", updateError)
	}

	return nil
}

func (userService *UserService) LoginWithDAuth(requestContext context.Context, loginRequest models.DAuthLoginRequest) (*models.User, error) {
	if loginRequest.Code == "" {
		return nil, errors.New("authorization code is required")
	}

	clientID := userService.config.DAuthClientID
	clientSecret := userService.config.DAuthClientSecret
	redirectURI := loginRequest.RedirectURI
	if redirectURI == "" {
		redirectURI = userService.config.DAuthRedirectURI
	}

	if clientID == "" || clientSecret == "" {
		return nil, errors.New("dauth credentials (client_id, client_secret) are not configured on server")
	}

	// 1. Exchange authorization code for access token
	tokenEndpoint := "https://auth.delta.nitt.edu/api/oauth/token"
	formData := url.Values{}
	formData.Set("client_id", clientID)
	formData.Set("client_secret", clientSecret)
	formData.Set("grant_type", "authorization_code")
	formData.Set("code", loginRequest.Code)
	formData.Set("redirect_uri", redirectURI)

	tokenHTTPRequest, reqErr := http.NewRequestWithContext(
		requestContext,
		http.MethodPost,
		tokenEndpoint,
		strings.NewReader(formData.Encode()),
	)
	if reqErr != nil {
		return nil, fmt.Errorf("failed to create dauth token request: %w", reqErr)
	}
	tokenHTTPRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	tokenHTTPResponse, tokenErr := userService.httpClient.Do(tokenHTTPRequest)
	if tokenErr != nil {
		return nil, fmt.Errorf("dauth token exchange network error: %w", tokenErr)
	}
	defer tokenHTTPResponse.Body.Close()

	tokenResponseBody, readErr := io.ReadAll(tokenHTTPResponse.Body)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read dauth token response: %w", readErr)
	}

	var tokenResult models.DAuthTokenResponse
	if unmarshalErr := json.Unmarshal(tokenResponseBody, &tokenResult); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to parse dauth token response: %w (body: %s)", unmarshalErr, string(tokenResponseBody))
	}

	if tokenResult.AccessToken == "" {
		errMsg := tokenResult.Error
		if tokenResult.ErrorDesc != "" {
			errMsg = fmt.Sprintf("%s: %s", errMsg, tokenResult.ErrorDesc)
		}
		if errMsg == "" {
			errMsg = string(tokenResponseBody)
		}
		return nil, fmt.Errorf("failed to retrieve access token from dauth: %s", errMsg)
	}

	// 2. Fetch User Profile from DAuth resource endpoint
	userResourceEndpoint := "https://auth.delta.nitt.edu/api/resources/user"
	userHTTPRequest, userReqErr := http.NewRequestWithContext(
		requestContext,
		http.MethodPost,
		userResourceEndpoint,
		nil,
	)
	if userReqErr != nil {
		return nil, fmt.Errorf("failed to create dauth user info request: %w", userReqErr)
	}
	userHTTPRequest.Header.Set("Authorization", "Bearer "+tokenResult.AccessToken)

	userHTTPResponse, userFetchErr := userService.httpClient.Do(userHTTPRequest)
	if userFetchErr != nil {
		return nil, fmt.Errorf("failed to fetch user info from dauth: %w", userFetchErr)
	}
	defer userHTTPResponse.Body.Close()

	// If POST fails with 405 Method Not Allowed, fallback to GET
	var userResponseBody []byte
	if userHTTPResponse.StatusCode == http.StatusMethodNotAllowed {
		getReq, _ := http.NewRequestWithContext(requestContext, http.MethodGet, userResourceEndpoint, nil)
		getReq.Header.Set("Authorization", "Bearer "+tokenResult.AccessToken)
		getResp, getErr := userService.httpClient.Do(getReq)
		if getErr == nil {
			defer getResp.Body.Close()
			userResponseBody, _ = io.ReadAll(getResp.Body)
		}
	} else {
		userResponseBody, readErr = io.ReadAll(userHTTPResponse.Body)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read dauth user response: %w", readErr)
		}
	}

	var dauthUser models.DAuthUserResponse
	if unmarshalErr := json.Unmarshal(userResponseBody, &dauthUser); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to parse dauth user info: %w (body: %s)", unmarshalErr, string(userResponseBody))
	}

	if dauthUser.Email == "" {
		return nil, fmt.Errorf("dauth did not return an email address: %s", string(userResponseBody))
	}

	// 3. Upsert user into database
	cleanEmail := strings.ToLower(strings.TrimSpace(dauthUser.Email))
	userName := strings.TrimSpace(dauthUser.Name)
	if userName == "" {
		userName = strings.Split(cleanEmail, "@")[0]
	}

	userUUID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("dauth:"+cleanEmail))
	pfpURL := fmt.Sprintf("https://api.dicebear.com/7.x/initials/svg?seed=%s", url.QueryEscape(userName))

	// Extract roll number: from dauthUser.RollNo or by stripping @nitt.edu / @... from cleanEmail
	rollNumber := strings.TrimSpace(dauthUser.RollNo)
	if rollNumber == "" && strings.Contains(cleanEmail, "@") {
		rollNumber = strings.TrimSuffix(cleanEmail, "@nitt.edu")
		if strings.Contains(rollNumber, "@") {
			rollNumber = strings.Split(rollNumber, "@")[0]
		}
		rollNumber = strings.TrimSpace(rollNumber)
	}

	// Extract & normalize gender from DAuth response
	var userGender *string
	if dauthUser.Gender != "" {
		g := strings.ToLower(strings.TrimSpace(dauthUser.Gender))
		if g == "m" || g == "male" {
			g = "male"
		} else if g == "f" || g == "female" {
			g = "female"
		} else {
			g = "other"
		}
		userGender = &g
	}

	// Personal email: Keep empty if it is a @nitt.edu email so user can enter personal email in the form
	var personalEmail string
	if !strings.HasSuffix(cleanEmail, "@nitt.edu") {
		personalEmail = cleanEmail
	}

	userModelToSave := &models.User{
		UserID: userUUID,
		Email:  personalEmail,
		Name:   userName,
		Pfp:    &pfpURL,
	}

	if rollNumber != "" {
		userModelToSave.RollNumber = &rollNumber
	}

	if userGender != nil {
		userModelToSave.Gender = userGender
	}

	savedUserRecord, upsertError := userService.userRepository.UpsertUser(requestContext, userModelToSave)
	if upsertError != nil {
		return nil, fmt.Errorf("failed to sync dauth user in database: %w", upsertError)
	}

	return savedUserRecord, nil
}

func (userService *UserService) EnsureMasterUser(requestContext context.Context) error {
	masterEmail := "anshul@gmail.com"
	existingUser, err := userService.userRepository.FindUserByEmail(requestContext, masterEmail)
	if err != nil {
		return fmt.Errorf("failed to check master user: %w", err)
	}

	rollNo := "112125003"
	gender := "male"
	pfp := "https://api.dicebear.com/7.x/initials/svg?seed=Anshul"
	name := "Anshul"

	if existingUser == nil {
		userUUID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("master:"+masterEmail))
		masterUser := &models.User{
			UserID:     userUUID,
			Name:       name,
			Email:      masterEmail,
			RollNumber: &rollNo,
			Gender:     &gender,
			Pfp:        &pfp,
		}
		_, createErr := userService.userRepository.UpsertUser(requestContext, masterUser)
		if createErr != nil {
			return fmt.Errorf("failed to create master user: %w", createErr)
		}
		fmt.Printf("Master user (%s) verified/created with roll number %s\n", masterEmail, rollNo)
	} else if existingUser.RollNumber == nil || *existingUser.RollNumber == "" {
		existingUser.RollNumber = &rollNo
		existingUser.Gender = &gender
		_, _ = userService.userRepository.UpsertUser(requestContext, existingUser)
	}

	return nil
}

func (userService *UserService) LoginWithEmail(requestContext context.Context, loginReq models.EmailLoginRequest) (*models.User, error) {
	cleanEmail := strings.ToLower(strings.TrimSpace(loginReq.Email))
	if cleanEmail == "" {
		return nil, errors.New("email is required")
	}

	// 1. If master user credentials
	if cleanEmail == "anshul@gmail.com" {
		if strings.TrimSpace(loginReq.Password) != "anshul" {
			return nil, errors.New("invalid password for master user")
		}
		_ = userService.EnsureMasterUser(requestContext)
		userRecord, err := userService.userRepository.FindUserByEmail(requestContext, cleanEmail)
		if err == nil && userRecord != nil {
			return userRecord, nil
		}
	}

	// 2. Only allow users who already exist in database
	existingUser, err := userService.userRepository.FindUserByEmail(requestContext, cleanEmail)
	if err != nil || existingUser == nil {
		return nil, errors.New("user not found in database. Only pre-registered users can sign in with password")
	}

	return existingUser, nil
}


