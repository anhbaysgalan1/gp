import React from "react";
import { AppProps } from "next/app";
import { AuthProvider } from "../contexts/AuthContext";
import { SessionProvider } from "../contexts/SessionContext";
import { AppStoreProvider } from "../providers/AppStore";
import { WebSocketProvider } from "../contexts/WebSocketContext";

import "../styles/index.css";

function MyApp({ Component, pageProps }: AppProps) {
    return (
        <AuthProvider>
            <SessionProvider>
                <AppStoreProvider>
                    <WebSocketProvider>
                        <Component {...pageProps} />
                    </WebSocketProvider>
                </AppStoreProvider>
            </SessionProvider>
        </AuthProvider>
    );
}

export default MyApp;
