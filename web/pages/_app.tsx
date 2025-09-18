import React from "react";
import { AppProps } from "next/app";
import { AuthProvider } from "../contexts/AuthContext";
import { SessionProvider } from "../contexts/SessionContext";
import { AppStoreProvider } from "../providers/AppStore";
import { SocketProvider } from "../providers/WebSocket";

import "../styles/index.css";

function MyApp({ Component, pageProps }: AppProps) {
    return (
        <AuthProvider>
            <SessionProvider>
                <AppStoreProvider>
                    <SocketProvider>
                        <Component {...pageProps} />
                    </SocketProvider>
                </AppStoreProvider>
            </SessionProvider>
        </AuthProvider>
    );
}

export default MyApp;
