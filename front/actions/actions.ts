import { wsManager, WebSocketMessage } from '../lib/websocket-manager';

// Helper function to send WebSocket messages safely
function sendWSMessage(message: WebSocketMessage): boolean {
    const sent = wsManager.sendMessage(message);
    if (!sent) {
        console.warn("WebSocket message queued:", message.action);
    }
    return sent;
}

export function joinTable(tablename: string): boolean {
    return sendWSMessage({
        action: "join-table",
        tablename: tablename,
    });
}

export function sendMessage(username: string, message: string): boolean {
    return sendWSMessage({
        action: "send-message",
        username: username,
        message: message,
    });
}

export function sendLog(message: string): boolean {
    return sendWSMessage({
        action: "send-log",
        message: message,
    });
}

export function takeSeat(username: string, seatID: number, buyIn: number): boolean {
    return sendWSMessage({
        action: "take-seat",
        username: username,
        seatID: seatID,
        buyIn: buyIn,
    });
}

export function leaveTable(tablename: string): boolean {
    return sendWSMessage({
        action: "leave-table",
        tablename: tablename,
    });
}

export function startGame(): boolean {
    return sendWSMessage({
        action: "start-game",
    });
}

export function resetGame(): boolean {
    return sendWSMessage({
        action: "reset-game",
    });
}

export function dealGame(): boolean {
    return sendWSMessage({
        action: "deal-game",
    });
}

export function newPlayer(username: string): boolean {
    return sendWSMessage({
        action: "new-player",
        username: username,
    });
}

export function playerCall(): boolean {
    return sendWSMessage({
        action: "player-call",
    });
}

export function playerCheck(): boolean {
    return sendWSMessage({
        action: "player-check",
    });
}

export function playerRaise(amount: number): boolean {
    return sendWSMessage({
        action: "player-raise",
        amount: amount,
    });
}

export function playerFold(): boolean {
    return sendWSMessage({
        action: "player-fold",
    });
}

export function getBalance(): boolean {
    return sendWSMessage({
        action: "get-balance",
    });
}
