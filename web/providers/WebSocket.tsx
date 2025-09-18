import { createContext, ReactChild, useEffect, useState, useContext } from "react";
import { AppContext } from "../providers/AppStore";
import { Message, Game, Log } from "../interfaces";
import { config } from "../lib/config";
import { useAuthContext } from "../contexts/AuthContext";
import { useSessionContext } from "../contexts/SessionContext";
import { SessionInfo } from "../types/api";
import { useRouter } from "next/router";

/*
WebSocket context creates a single connection to the server per client.
It handles opening, closing, and error handling of the websocket. It also
dispatches websocket messages to update the central state store.
*/

export const SocketContext = createContext<WebSocket | null>(null);

type SocketProviderProps = {
    children: ReactChild;
};

export function SocketProvider(props: SocketProviderProps) {
    const [socket, setSocket] = useState<WebSocket | null>(null);
    const { appState, dispatch } = useContext(AppContext);
    const { user } = useAuthContext();
    const { setSessionInfo } = useSessionContext();
    const router = useRouter();

    useEffect(() => {
        // WebSocket api is browser side only.
        const isBrowser = typeof window !== "undefined";

        let wsUrl = '';
        if (isBrowser && user) {
            // Get JWT token for authentication
            const token = localStorage.getItem('auth_token');
            if (!token) {
                console.warn("No auth token found, WebSocket connection will fail");
                return;
            }

            // Use backend WebSocket URL from configuration with auth token
            wsUrl = `${config.websocket.url}/ws?token=${encodeURIComponent(token)}`;
            console.log("websocket url: ", wsUrl);
        }

        const _socket = isBrowser && user ? new WebSocket(wsUrl) : null;

        if (_socket) {
            _socket.onopen = () => {
                console.log("websocket connected");

                // Set username from auth context
                if (user && user.username) {
                    dispatch({ type: "setUsername", payload: user.username });
                }

                // Auto-join table if user is on a game page
                if (router.pathname.startsWith('/game/') && router.query.tableId) {
                    const tableId = router.query.tableId as string;
                    console.log("Auto-joining table:", tableId);
                    _socket.send(JSON.stringify({
                        action: "join-table",
                        tablename: tableId
                    }));

                    // Set username in app state if available
                    if (user && user.username) {
                        dispatch({ type: "setUsername", payload: user.username });
                    }

                    // Request current game state on reconnection (after a small delay)
                    setTimeout(() => {
                        if (_socket.readyState === WebSocket.OPEN) {
                            _socket.send(JSON.stringify({
                                action: "get-balance"
                            }));
                        }
                    }, 200);
                }
            };
            _socket.onclose = () => {
                console.log("websocket disconnected");
            };
            _socket.onerror = (error) => {
                console.error("websocket error: ", error);
            };
        }
        // Set up WebSocket message handler inside useEffect to prevent duplicates
        if (_socket) {
            _socket.onmessage = (e) => {
                let event = JSON.parse(e.data);
                switch (event.action) {
                    case "new-message":
                        let newMessage: Message = {
                            name: event.username,
                            message: event.message,
                            timestamp: event.timestamp,
                        };
                        dispatch({ type: "addMessage", payload: newMessage });
                        return;
                    case "new-log":
                        let newLog: Log = {
                            message: event.message,
                            timestamp: event.timestamp,
                        };
                        dispatch({ type: "addLog", payload: newLog });
                        return;
                    case "update-game":
                        let newGame: Game = {
                            running: event.game.running,
                            dealer: event.game.dealer,
                            action: event.game.action,
                            utg: event.game.utg,
                            sb: event.game.sb,
                            bb: event.game.bb,
                            communityCards: event.game.communityCards,
                            stage: event.game.stage,
                            betting: event.game.betting,
                            config: event.game.config,
                            players: event.game.players,
                            pots: event.game.pots,
                            minRaise: event.game.minRaise,
                            readyCount: event.game.readyCount,
                        };
                        dispatch({ type: "updateGame", payload: newGame });

                        // Handle session info update from WebSocket
                        if (event.session_info) {
                            setSessionInfo(event.session_info);
                            console.log("Session info updated:", event.session_info);
                        } else {
                            // Clear session info if not provided (user not authenticated or no session)
                            setSessionInfo(null);
                        }
                        return;
                    case "update-player-uuid":
                        dispatch({ type: "updatePlayerID", payload: event.uuid });
                        return;
                    case "error":
                        // Show error message to user
                        console.error("WebSocket error:", event.message);
                        alert(event.message); // TODO: Replace with proper toast notification
                        return;
                    case "success":
                        // Show success message to user
                        console.log("WebSocket success:", event.message);
                        // TODO: Replace with proper toast notification
                        return;
                    case "update-balance":
                        // Handle real-time balance updates
                        const balanceUpdate = {
                            main_balance: event.main_balance,
                            game_balance: event.game_balance,
                            total_balance: event.main_balance + event.game_balance,
                        };
                        console.log("Balance update received:", balanceUpdate);

                        // Trigger SWR cache update for balance
                        // We'll emit a custom event that the useBalance hook can listen to
                        if (typeof window !== 'undefined') {
                            window.dispatchEvent(new CustomEvent('balance-update', {
                                detail: balanceUpdate
                            }));
                        }
                        return;
                    default:
                        console.warn("Unknown WebSocket message:", event);
                        return;
                }
            };
        }

        setSocket(_socket);

        // Listen for balance update requests from useBalance hook
        const handleBalanceRequest = () => {
            if (_socket && _socket.readyState === WebSocket.OPEN) {
                _socket.send(JSON.stringify({ action: "get-balance" }));
            }
        };

        if (isBrowser) {
            window.addEventListener('request-balance', handleBalanceRequest);
        }

        return () => {
            socket?.close();
            if (isBrowser) {
                window.removeEventListener('request-balance', handleBalanceRequest);
            }
        };
    }, [dispatch, user, router.pathname, router.query.tableId, setSessionInfo]);

    return <SocketContext.Provider value={socket}>{props.children}</SocketContext.Provider>;
}
