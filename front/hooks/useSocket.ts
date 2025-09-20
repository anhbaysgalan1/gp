import { useContext, useEffect, useRef, useState } from "react";
import { io, Socket } from "socket.io-client";
import { AppContext } from "../providers/AppStore";

const WS_URL = process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:8080";

export function useSocket() {
    const [socket, setSocket] = useState<Socket | null>(null);
    const { appState, dispatch } = useContext(AppContext);
    const socketRef = useRef<Socket | null>(null);

    useEffect(() => {
        if (!socketRef.current) {
            const newSocket = io(WS_URL, {
                transports: ["websocket"],
                upgrade: false,
            });

            newSocket.on("connect", () => {
                console.log("Connected to server");
            });

            newSocket.on("disconnect", () => {
                console.log("Disconnected from server");
            });

            newSocket.on("error", (error) => {
                console.error("Socket error:", error);
            });

            socketRef.current = newSocket;
            setSocket(newSocket);
        }

        return () => {
            if (socketRef.current) {
                socketRef.current.disconnect();
                socketRef.current = null;
            }
        };
    }, []);

    return socket;
}