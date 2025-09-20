import React, { useState, useContext, useCallback, useEffect } from "react";
import { takeSeat, sendLog } from "../actions/actions";
import { useWebSocket } from "../contexts/WebSocketContext";
import { AppContext } from "../providers/AppStore";
import { FcCheckmark } from "react-icons/fc";
import { useBalance } from "../hooks/useApi";
import { formatMNT } from "../lib/api-utils";
import { useRouter } from "next/router";
import { useSessionContext } from "../contexts/SessionContext";

type buyInProps = {
    seatID: number;
    sitDown: boolean;
    setSitDown: React.Dispatch<React.SetStateAction<boolean>>;
};

export default function BuyIn({ seatID, sitDown, setSitDown }: buyInProps) {
    const { isConnected } = useWebSocket();
    const { appState, dispatch } = useContext(AppContext);
    const { balance, isLoading: balanceLoading } = useBalance();
    const { hasSession } = useSessionContext();
    const router = useRouter();
    const [buyIn, setBuyIn] = useState(
        appState.game?.config.maxBuyIn ? appState.game?.config.maxBuyIn : 2000
    );
    const [isProcessing, setIsProcessing] = useState(false);

    // Close the modal when session is established (user successfully took seat)
    useEffect(() => {
        if (hasSession && isProcessing) {
            setIsProcessing(false);
            setSitDown(false);
        }
    }, [hasSession, isProcessing, setSitDown]);

    const handleBuyIn = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        const amount = parseInt(e.target.value);
        setBuyIn(amount);
    }, []);

    const handleSitDown = () => {
        if (!isConnected || !appState.username) {
            alert("Please ensure you are connected and have entered a username");
            return;
        }

        // Check if balance is loaded
        if (balanceLoading) {
            alert("Loading balance, please wait...");
            return;
        }

        // Check for sufficient balance
        if (balance && balance.main_balance < buyIn) {
            const confirmed = confirm(
                `Insufficient balance for buy-in. You have ${formatMNT(balance.main_balance)} but need ${formatMNT(buyIn)}. Would you like to go to your wallet to add more funds?`
            );
            if (confirmed) {
                router.push('/wallet');
            }
            return;
        }

        setIsProcessing(true);

        try {
            const seatSuccess = takeSeat(appState.username, seatID, buyIn);
            const logMessage = appState.username + " is attempting to buy in for " + formatMNT(buyIn);
            const logSuccess = sendLog(logMessage);

            if (!seatSuccess || !logSuccess) {
                console.warn("WebSocket messages queued due to connection state");
            }

            // Modal will close automatically via useEffect when hasSession becomes true
        } catch (error) {
            console.error("Error taking seat:", error);
            alert("Failed to take seat. Please try again.");
            setIsProcessing(false);
        }
    };
    return (
        <div className="relative right-1 m-4 flex h-full w-full flex-col items-start justify-center">
            <p className="-mb-1 text-lg font-semibold">{appState.username}</p>

            {/* Balance Display */}
            <div className="mb-2 text-sm">
                {balanceLoading ? (
                    <p className="text-gray-400">Loading balance...</p>
                ) : balance ? (
                    <p className="text-green-400">
                        Available: {formatMNT(balance.main_balance)}
                    </p>
                ) : (
                    <p className="text-red-400">Unable to load balance</p>
                )}
            </div>

            <div className="flex flex-row items-center">
                <p>Buy In: </p>
                <input
                    autoFocus
                    className="ml-4 mr-2 w-20 rounded-sm bg-neutral-500 p-1 text-white focus:outline-none"
                    id="buyIn"
                    type="number"
                    value={buyIn}
                    onChange={handleBuyIn}
                    disabled={isProcessing}
                />
                <button
                    onClick={handleSitDown}
                    className={`text-2xl ${isProcessing ? 'opacity-50 cursor-not-allowed' : ''}`}
                    disabled={isProcessing}
                >
                    {isProcessing ? '⏳' : <FcCheckmark />}
                </button>
            </div>

            {/* Validation Messages */}
            {balance && balance.main_balance < buyIn && (
                <p className="mt-1 text-xs text-red-400">
                    Insufficient balance. Need {formatMNT(buyIn - balance.main_balance)} more.
                </p>
            )}
        </div>
    );
}
