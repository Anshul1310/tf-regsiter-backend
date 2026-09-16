package handlers

import (
	"strings"

	"github.com/Anshul1310/tf-register/internals/models"
	"github.com/Anshul1310/tf-register/internals/service"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type TeamHandler struct {
	teamService     *service.TeamService
	cashfreeService *service.CashfreeService
}

func NewTeamHandler(
	teamService *service.TeamService,
	cashfreeService *service.CashfreeService,
) *TeamHandler {
	return &TeamHandler{
		teamService:     teamService,
		cashfreeService: cashfreeService,
	}
}

func (teamHandler *TeamHandler) CreateTeam(requestContext *fiber.Ctx) error {
	authenticatedUserValue := requestContext.Locals("userID")
	var leaderUserID uuid.UUID
	if authenticatedUserValue != nil {
		leaderUserID = authenticatedUserValue.(uuid.UUID)
	}

	var leaderEmail string
	authenticatedEmailValue := requestContext.Locals("userEmail")
	if authenticatedEmailValue != nil {
		leaderEmail = authenticatedEmailValue.(string)
	}

	var createTeamRequest models.CreateTeamRequest
	bodyParseError := requestContext.BodyParser(&createTeamRequest)
	if bodyParseError != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Failed to parse create team request: " + bodyParseError.Error(),
		})
	}

	// Fallback to request values if provided
	if leaderUserID == uuid.Nil && createTeamRequest.LeaderUserID != uuid.Nil {
		leaderUserID = createTeamRequest.LeaderUserID
	}
	if leaderEmail == "" && createTeamRequest.LeaderEmail != "" {
		leaderEmail = createTeamRequest.LeaderEmail
	}

	if leaderUserID == uuid.Nil {
		return requestContext.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "User authentication required to create a team",
		})
	}

	createdTeamRecord, creationError := teamHandler.teamService.CreateTeam(
		requestContext.Context(),
		leaderUserID,
		leaderEmail,
		createTeamRequest,
	)
	if creationError != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": creationError.Error(),
		})
	}

	return requestContext.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Team created successfully",
		"data":    createdTeamRecord,
	})
}

func (teamHandler *TeamHandler) JoinTeam(requestContext *fiber.Ctx) error {
	authenticatedUserValue := requestContext.Locals("userID")
	if authenticatedUserValue == nil {
		return requestContext.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Authentication required to join a team",
		})
	}

	authenticatedUserID := authenticatedUserValue.(uuid.UUID)

	var joinTeamRequest models.JoinTeamRequest
	bodyParseError := requestContext.BodyParser(&joinTeamRequest)
	if bodyParseError != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Failed to parse join team request: " + bodyParseError.Error(),
		})
	}

	joinError := teamHandler.teamService.JoinTeam(
		requestContext.Context(),
		authenticatedUserID,
		joinTeamRequest.TeamCode,
	)
	if joinError != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": joinError.Error(),
		})
	}

	return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Successfully joined the team",
	})
}

func (teamHandler *TeamHandler) GetTeamByID(requestContext *fiber.Ctx) error {
	teamIdentifierParameter := requestContext.Params("id")
	if teamIdentifierParameter == "" {
		teamIdentifierParameter = requestContext.Params("teamId")
	}

	teamRecord, findError := teamHandler.teamService.GetTeamDetails(requestContext.Context(), teamIdentifierParameter)
	if findError != nil {
		return requestContext.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": findError.Error(),
		})
	}

	// Restrict team details to members & leader only
	authenticatedUserValue := requestContext.Locals("userID")
	if authenticatedUserValue != nil {
		authUserID := authenticatedUserValue.(uuid.UUID)
		isMemberOrLeader := teamRecord.LeaderUserID == authUserID
		if !isMemberOrLeader {
			for _, member := range teamRecord.Members {
				if member.UserID == authUserID {
					isMemberOrLeader = true
					break
				}
			}
		}
		if !isMemberOrLeader {
			return requestContext.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"message": "Access restricted: You must be a member of this team to view its details",
			})
		}
	}

	return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    teamRecord,
	})
}

func (teamHandler *TeamHandler) GetAllPublicTeams(requestContext *fiber.Ctx) error {
	domainFilterQuery := requestContext.Query("domain")
	searchQuery := requestContext.Query("search")

	publicTeamsList, queryError := teamHandler.teamService.GetPublicTeams(
		requestContext.Context(),
		domainFilterQuery,
		searchQuery,
	)
	if queryError != nil {
		return requestContext.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": queryError.Error(),
		})
	}

	return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    publicTeamsList,
	})
}

func (teamHandler *TeamHandler) GetMemberCount(requestContext *fiber.Ctx) error {
	teamIdentifierParameter := requestContext.Params("id")
	if teamIdentifierParameter == "" {
		teamIdentifierParameter = requestContext.Params("teamId")
	}

	memberCount, countError := teamHandler.teamService.GetTeamMemberCount(requestContext.Context(), teamIdentifierParameter)
	if countError != nil {
		return requestContext.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": countError.Error(),
		})
	}

	return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"count":   memberCount,
		"data":    memberCount,
	})
}

func (teamHandler *TeamHandler) GetPaymentStats(requestContext *fiber.Ctx) error {
	paidTeamCount, queryError := teamHandler.teamService.GetPaidTeamCount(requestContext.Context())
	if queryError != nil {
		return requestContext.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": queryError.Error(),
		})
	}

	return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"count":   paidTeamCount,
	})
}

func (teamHandler *TeamHandler) UpdateVisibility(requestContext *fiber.Ctx) error {
	authenticatedUserValue := requestContext.Locals("userID")
	if authenticatedUserValue == nil {
		return requestContext.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Authentication required",
		})
	}

	authenticatedUserID := authenticatedUserValue.(uuid.UUID)
	teamIdentifier := requestContext.Params("teamId")
	if teamIdentifier == "" {
		teamIdentifier = requestContext.Params("id")
	}

	var visibilityRequest models.UpdateVisibilityRequest
	bodyParseError := requestContext.BodyParser(&visibilityRequest)
	if bodyParseError != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Failed to parse visibility request: " + bodyParseError.Error(),
		})
	}

	updateError := teamHandler.teamService.UpdateVisibility(
		requestContext.Context(),
		authenticatedUserID,
		teamIdentifier,
		visibilityRequest.IsPublic,
	)
	if updateError != nil {
		return requestContext.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"message": updateError.Error(),
		})
	}

	return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Team visibility updated successfully",
	})
}

func (teamHandler *TeamHandler) UpdateTeamName(requestContext *fiber.Ctx) error {
	authenticatedUserValue := requestContext.Locals("userID")
	if authenticatedUserValue == nil {
		return requestContext.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Authentication required",
		})
	}

	authenticatedUserID := authenticatedUserValue.(uuid.UUID)
	teamIdentifier := requestContext.Params("teamId")
	if teamIdentifier == "" {
		teamIdentifier = requestContext.Params("id")
	}

	var nameUpdateRequest models.UpdateTeamNameRequest
	bodyParseError := requestContext.BodyParser(&nameUpdateRequest)
	if bodyParseError != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Failed to parse name update request: " + bodyParseError.Error(),
		})
	}

	updateError := teamHandler.teamService.UpdateTeamName(
		requestContext.Context(),
		authenticatedUserID,
		teamIdentifier,
		nameUpdateRequest.Name,
	)
	if updateError != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": updateError.Error(),
		})
	}

	return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Team name updated successfully",
	})
}

func (teamHandler *TeamHandler) UpdateTeamDetails(requestContext *fiber.Ctx) error {
	authenticatedUserValue := requestContext.Locals("userID")
	if authenticatedUserValue == nil {
		return requestContext.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Authentication required",
		})
	}

	authenticatedUserID := authenticatedUserValue.(uuid.UUID)
	teamIdentifier := requestContext.Params("teamId")
	if teamIdentifier == "" {
		teamIdentifier = requestContext.Params("id")
	}

	var detailsUpdateRequest models.UpdateTeamDetailsRequest
	bodyParseError := requestContext.BodyParser(&detailsUpdateRequest)
	if bodyParseError != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Failed to parse details update request: " + bodyParseError.Error(),
		})
	}

	updateError := teamHandler.teamService.UpdateTeamDetails(
		requestContext.Context(),
		authenticatedUserID,
		teamIdentifier,
		detailsUpdateRequest.Domain,
		detailsUpdateRequest.ProblemStatement,
	)
	if updateError != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": updateError.Error(),
		})
	}

	return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Team details updated successfully",
	})
}

func (teamHandler *TeamHandler) RegenerateTeamID(requestContext *fiber.Ctx) error {
	authenticatedUserValue := requestContext.Locals("userID")
	if authenticatedUserValue == nil {
		return requestContext.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Authentication required",
		})
	}

	authenticatedUserID := authenticatedUserValue.(uuid.UUID)
	teamIdentifier := requestContext.Params("teamId")
	if teamIdentifier == "" {
		teamIdentifier = requestContext.Params("id")
	}

	newTeamID, regenerateError := teamHandler.teamService.RegenerateTeamID(
		requestContext.Context(),
		authenticatedUserID,
		teamIdentifier,
	)
	if regenerateError != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": regenerateError.Error(),
		})
	}

	return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":     true,
		"message":     "Team ID regenerated successfully",
		"new_team_id": newTeamID,
	})
}

func (teamHandler *TeamHandler) DeleteTeam(requestContext *fiber.Ctx) error {
	authenticatedUserValue := requestContext.Locals("userID")
	if authenticatedUserValue == nil {
		return requestContext.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Authentication required",
		})
	}

	authenticatedUserID := authenticatedUserValue.(uuid.UUID)
	teamIdentifier := requestContext.Params("teamId")
	if teamIdentifier == "" {
		teamIdentifier = requestContext.Params("id")
	}

	deleteError := teamHandler.teamService.DeleteTeam(
		requestContext.Context(),
		authenticatedUserID,
		teamIdentifier,
	)
	if deleteError != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": deleteError.Error(),
		})
	}

	return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Team deleted successfully",
	})
}

func (teamHandler *TeamHandler) SubmitManualPayment(requestContext *fiber.Ctx) error {
	teamIdentifier := requestContext.Params("teamId")
	transactionID := requestContext.FormValue("transactionId")
	screenshotURL := ""

	// Check if file was uploaded
	uploadedFile, fileError := requestContext.FormFile("screenshot")
	if fileError == nil && uploadedFile != nil {
		screenshotURL = "/uploads/" + uploadedFile.Filename
		// Save uploaded screenshot if uploads directory exists
		_ = requestContext.SaveFile(uploadedFile, "./uploads/"+uploadedFile.Filename)
	}

	submissionError := teamHandler.teamService.SubmitManualPayment(
		requestContext.Context(),
		teamIdentifier,
		transactionID,
		screenshotURL,
	)
	if submissionError != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": submissionError.Error(),
		})
	}

	return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Payment submitted successfully and is currently processing",
	})
}

// Cashfree Endpoints

func (teamHandler *TeamHandler) CreateCheckoutOrder(requestContext *fiber.Ctx) error {
	var checkoutRequest models.CashfreeCheckoutRequest
	bodyParseError := requestContext.BodyParser(&checkoutRequest)
	if bodyParseError != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Failed to parse checkout request: " + bodyParseError.Error(),
		})
	}

	if strings.TrimSpace(checkoutRequest.TeamID) == "" {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "teamId is required",
		})
	}

	cashfreeOrderResponse, orderError := teamHandler.cashfreeService.CreatePaymentOrder(
		requestContext.Context(),
		checkoutRequest.TeamID,
		checkoutRequest.UserID,
		checkoutRequest.TeamName,
	)
	if orderError != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": orderError.Error(),
		})
	}

	// Returns exact JSON schema expected by frontend Dashboard.tsx (Order interface with payment_session_id)
	return requestContext.Status(fiber.StatusOK).JSON(cashfreeOrderResponse)
}

func (teamHandler *TeamHandler) VerifyPayment(requestContext *fiber.Ctx) error {
	var verifyRequest models.CashfreeVerifyRequest
	bodyParseError := requestContext.BodyParser(&verifyRequest)
	if bodyParseError != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Failed to parse verification request: " + bodyParseError.Error(),
		})
	}

	isPaid, verificationError := teamHandler.cashfreeService.VerifyPaymentOrder(
		requestContext.Context(),
		verifyRequest.OrderID,
		verifyRequest.TeamID,
	)
	if verificationError != nil {
		return requestContext.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": verificationError.Error(),
		})
	}

	return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"paid":    isPaid,
	})
}

func (teamHandler *TeamHandler) HandleCashfreeWebhook(requestContext *fiber.Ctx) error {
	webhookBodyBytes := requestContext.Body()
	processError := teamHandler.cashfreeService.ProcessWebhook(requestContext.Context(), webhookBodyBytes)
	if processError != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": processError.Error(),
		})
	}

	return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Webhook processed successfully",
	})
}

func (teamHandler *TeamHandler) ApplyToJoinTeam(requestContext *fiber.Ctx) error {
	authenticatedUserValue := requestContext.Locals("userID")
	if authenticatedUserValue == nil {
		return requestContext.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Authentication required to apply to a team",
		})
	}
	authenticatedUserID := authenticatedUserValue.(uuid.UUID)

	var applyReq models.ApplyTeamRequest
	if err := requestContext.BodyParser(&applyReq); err != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body: " + err.Error(),
		})
	}

	if applyReq.TeamID == "" {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "team_id is required",
		})
	}

	applyErr := teamHandler.teamService.ApplyToJoinPublicTeam(
		requestContext.Context(),
		authenticatedUserID,
		applyReq.TeamID,
	)
	if applyErr != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": applyErr.Error(),
		})
	}

	return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Application to join team submitted successfully. Pending leader approval.",
	})
}

func (teamHandler *TeamHandler) GetTeamJoinRequests(requestContext *fiber.Ctx) error {
	teamIdentifier := requestContext.Params("teamId")
	if teamIdentifier == "" {
		teamIdentifier = requestContext.Params("id")
	}

	requests, err := teamHandler.teamService.GetPendingJoinRequests(requestContext.Context(), teamIdentifier)
	if err != nil {
		return requestContext.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    requests,
	})
}

func (teamHandler *TeamHandler) AcceptJoinRequest(requestContext *fiber.Ctx) error {
	authenticatedUserValue := requestContext.Locals("userID")
	if authenticatedUserValue == nil {
		return requestContext.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Authentication required",
		})
	}
	leaderUserID := authenticatedUserValue.(uuid.UUID)

	teamIdentifier := requestContext.Params("teamId")
	requestIDStr := requestContext.Params("requestId")
	requestID, parseErr := uuid.Parse(requestIDStr)
	if parseErr != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request ID",
		})
	}

	acceptErr := teamHandler.teamService.AcceptJoinRequest(
		requestContext.Context(),
		leaderUserID,
		teamIdentifier,
		requestID,
	)
	if acceptErr != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": acceptErr.Error(),
		})
	}

	return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Join request accepted successfully. Member has been added to your team.",
	})
}

func (teamHandler *TeamHandler) RejectJoinRequest(requestContext *fiber.Ctx) error {
	authenticatedUserValue := requestContext.Locals("userID")
	if authenticatedUserValue == nil {
		return requestContext.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Authentication required",
		})
	}
	leaderUserID := authenticatedUserValue.(uuid.UUID)

	teamIdentifier := requestContext.Params("teamId")
	requestIDStr := requestContext.Params("requestId")
	requestID, parseErr := uuid.Parse(requestIDStr)
	if parseErr != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request ID",
		})
	}

	rejectErr := teamHandler.teamService.RejectJoinRequest(
		requestContext.Context(),
		leaderUserID,
		teamIdentifier,
		requestID,
	)
	if rejectErr != nil {
		return requestContext.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": rejectErr.Error(),
		})
	}

	return requestContext.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Join request declined.",
	})
}

