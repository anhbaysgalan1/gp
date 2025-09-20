import { useContext, Dispatch } from "react";
import { resetGame } from "../actions/actions";
import { useWebSocket } from "../contexts/WebSocketContext";
import { AppContext } from "../providers/AppStore";

function handleResetGame(isConnected: boolean, dispatch: Dispatch<any>) {
    if (isConnected) {
        resetGame();
        dispatch({ type: "resetGame" });
    }
}
export default function Reset() {
    const { isConnected } = useWebSocket();
    const { appState, dispatch } = useContext(AppContext);

    return (
        <button
            className="m-2 p-2 text-zinc-800 hover:text-zinc-700"
            onClick={() => handleResetGame(isConnected, dispatch)}
        >
            reset
        </button>
    );
}
