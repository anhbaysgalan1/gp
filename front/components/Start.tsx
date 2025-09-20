import { useContext } from "react";
import { useWebSocket } from "../contexts/WebSocketContext";
import { AppContext } from "../providers/AppStore";
import { Player } from "../interfaces/index";

type startProps = {
    players: (Player | null)[];
};

function handleStartGame(sendMessage: ((message: any) => void) | null) {
    if (sendMessage) {
        sendMessage({ action: "start-game" });
    }
}

export default function Start({ players }: startProps) {
    const { sendMessage } = useWebSocket();
    const { appState, dispatch } = useContext(AppContext);
    const game = appState.game;
    const readyPlayers = players.filter((player) => player != null);

    if (!game) {
        return null;
    }

    if (!game.running && readyPlayers.length < 2) {
        return (
            <div
                className=" m-10 rounded-sm border border-2 border-neutral-400 p-2 px-4 py-2 text-2xl font-light text-neutral-300 opacity-10"
                title="Must have 2 or more players to start game"
            >
                Start
            </div>
        );
    }

    if (!game.running && readyPlayers.length >= 2) {
        return (
            <button
                className=" m-10 rounded-sm border border-2 border-neutral-400 p-2 px-4 py-2 text-2xl font-normal text-neutral-300 hover:underline"
                onClick={() => handleStartGame(sendMessage)}
            >
                Start
            </button>
        );
    }

    return null;
}
