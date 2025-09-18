package server

import (
	"github.com/evanofslack/go-poker/poker"
)

// inbound (client) actions
const (
	actionJoinTable    string = "join-table"
	actionLeaveTable   string = "leave-table"
	actionSendMessage  string = "send-message"
	actionSendLog      string = "send-log"
	actionNewPlayer    string = "new-player"
	actionTakeSeat     string = "take-seat"
	actionStartGame    string = "start-game"
	actionDealGame     string = "deal-game"
	actionResetGame    string = "reset-game"
	actionPlayerCall   string = "player-call"
	actionPlayerCheck  string = "player-check"
	actionPlayerRaise  string = "player-raise"
	actionPlayerFold   string = "player-fold"
	actionGetBalance   string = "get-balance"
)

type base struct {
	// allows for correctly identifying messages
	Action string `json:"action"`
}

type joinTable struct {
	base             // actionJoinTable
	Tablename string `json:"tablename"`
}

type leaveTable struct {
	base             // actionLeaveTable
	Tablename string `json:"tablename"`
}

type sendMessage struct {
	base            // actionSendMessage
	Username string `json:"username"`
	Message  string `json:"message"`
}

type sendLog struct {
	base           // actionSendLog
	Message string `json:"message"`
}

type newPlayer struct {
	base            // actionNewPlayer
	Username string `json:"username"`
}

type takeSeat struct {
	base            // actionTakeSeat
	Username string `json:"username"`
	SeatID   uint   `json:"seatID"`
	BuyIn    uint   `json:"buyIn"`
}

type startGame struct {
	base // actionStartGame
}

type resetGame struct {
	base // actionResetGame
}

type dealGame struct {
	base // actionDealGame
}

type playerCall struct {
	base // actionPlayerCall
}

type playerCheck struct {
	base // actionPlayerCheck
}

type playerRaise struct {
	base        // actionPlayerRaise
	Amount uint `json:"amount"`
}

type playerFold struct {
	base // actionPlayerFold
}

type getBalance struct {
	base // actionGetBalance
}

// outbound (server) actions
const (
	actionNewMessage       string = "new-message"
	actionNewLog           string = "new-log"
	actionUpdateGame       string = "update-game"
	actionUpdatePlayerUUID string = "update-player-uuid"
	actionUpdateBalance    string = "update-balance"
)

type newMessage struct {
	base             // actionNewMessage
	Id        string `json:"uuid"`
	Message   string `json:"message"`
	Username  string `json:"username"`
	Timestamp string `json:"timestamp"`
}

type newLog struct {
	base             // actionNewLog
	Id        string `json:"uuid"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

type updateGame struct {
	base                 // actionUpdateGame
	Game *poker.GameView `json:"game"`
	SessionInfo *SessionInfo `json:"session_info,omitempty"`
}

type SessionInfo struct {
	UserID       string `json:"user_id"`
	SessionID    string `json:"session_id,omitempty"`
	SeatNumber   *int   `json:"seat_number,omitempty"`
	IsSeated     bool   `json:"is_seated"`
	HasSession   bool   `json:"has_session"`
}

type updatePlayerUUID struct {
	base        //actionUpdatePlayerUUID
	Uuid string `json:"uuid"`
}

type updateBalance struct {
	base                    // actionUpdateBalance
	MainBalance    int64    `json:"main_balance"`
	GameBalance    int64    `json:"game_balance"`
	Currency       string   `json:"currency"`
	TransactionID  string   `json:"transaction_id,omitempty"`
	ChangeAmount   int64    `json:"change_amount,omitempty"`
	ChangeType     string   `json:"change_type,omitempty"` // "buy_in", "win", "cash_out", "transfer_in", "transfer_out"
	Timestamp      string   `json:"timestamp"`
}
