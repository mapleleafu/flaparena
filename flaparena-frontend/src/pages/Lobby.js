import React, { useState, useEffect, useRef } from 'react';
import backgroundImageSrc from '../assets/images/background.png';

const BASEURL = "localhost:8000";

const Lobby = () => {
    const [users, setUsers] = useState([]);
    const [isLoggedIn, setIsLoggedIn] = useState(false);
    const wsRef = useRef(null);

    const handleLoginSubmit = async (event) => {
        event.preventDefault();

        // Extract form data
        const formData = new FormData(event.target);
        const username = formData.get('username');
        const password = formData.get('password');

        try {
            const response = await apiCall(`http://${BASEURL}/api/login`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username, password }),
                credentials: 'include'
            });
            if (response.success === true) {
                const accessToken = response.data.access_token;
                localStorage.setItem('access_token', accessToken);
                setIsLoggedIn(true);
                setupWebSocket(accessToken);
            } else {
                console.error(response.message);
            }
        } catch (error) {
            console.error(error.message);
        }
    };

    const attemptLoginOrRefresh = async () => {
        let accessToken = localStorage.getItem('access_token');
    
        // Try to get the refresh token from cookies
        const refreshTokenCookie = document.cookie.split('; ').find(row => row.startsWith('refresh_token='));
        const refreshToken = refreshTokenCookie ? refreshTokenCookie.split('=')[1] : null;
    
        if (!accessToken && refreshToken) {
            // If there's a refresh token but no access token, try to refresh it
            try {
                const response = await fetch(`${BASEURL}/refresh/token`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    }
                });
    
                if (!response.ok) throw new Error("Refresh token invalid");
    
                accessToken = response.data.access_token;
    
                // Save the new access token to local storage
                localStorage.setItem('access_token', accessToken);
                setIsLoggedIn(true);
                return accessToken;
    
            } catch (error) {
                console.error("Login or token refresh failed:", error);
                setIsLoggedIn(false);
            }
        } else if (!accessToken) {
            // Handle the case where there's neither an access token nor a refresh token
            console.log("Please log in");
            setIsLoggedIn(false);
        } else {
            // Access token is available
            setIsLoggedIn(true);
            return accessToken;
        }
    };

    useEffect(() => {
        (async () => {
            const accessToken = await attemptLoginOrRefresh();
            if (accessToken !== undefined) {
                setupWebSocket(accessToken);
            }
        })();
    }, []);

    const setupWebSocket = (accessToken) => {
        console.log("Trying to set up WebSocket");
        wsRef.current = new WebSocket(`ws://${BASEURL}/ws/${accessToken}`);

        wsRef.current.onopen = () => {
            console.log("Connected to the lobby");
        };

        wsRef.current.onmessage = (event) => {
            const message = JSON.parse(event.data);
            switch (message.type) {
                case 'gameState':
                    console.log(message);
                    setUsers(message.data.map(user => ({
                        id: user.userID,
                        username: user.username,
                        connected: user.connected,
                        ready: user.ready
                    })));
                    break;
                default:
                    console.log("Received message: ", message);
            }
        };

        wsRef.current.onerror = (error) => {
            console.log("WebSocket Error: ", error);
        };

        wsRef.current.onclose = (event) => {
            console.log("Disconnected from the lobby", event.code, event.reason);
        };
    };

    const sendReady = () => {
        const message = JSON.stringify({ action: "ready", timestamp: Date.now() });
        wsRef.current?.send(message);

        // Update the local state immediately
        setUsers(users.map(user => ({
            ...user,
            ready: true
        })));
    };

    const lobbyInfo = () => {
        const message = JSON.stringify({ action: "info", timestamp: Date.now() });
        wsRef.current?.send(message);
    }

    if (!isLoggedIn) {
        return (
            <div style={{ backgroundImage: `url(${backgroundImageSrc})`, height: '100vh', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center' }}>
                <h1>Log in to play</h1>
                <form onSubmit={handleLoginSubmit}>
                    <label htmlFor="username">Username:</label>
                    <input type="text" id="username" name="username" required />
                    <label htmlFor="password">Password:</label>
                    <input type="password" id="password" name="password" required />
                    <button type="submit">Log in</button>
                </form>
            </div>
        );
    }

    return (
        <div style={{ backgroundImage: `url(${backgroundImageSrc})`, height: '100vh', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center' }}>
            <h1>Lobby</h1>
            <ul>
                {users.map((user) => (
                    <li key={user.id}>
                        {user.username} - {user.connected ? 'Connected' : 'Disconnected'} - {user.ready ? 'Ready' : 'Not Ready'}
                    </li>
                ))}
            </ul>
            <button onClick={sendReady}>I'm Ready</button>
            <button onClick={lobbyInfo}>Lobby Info</button>
        </div>
    );
};

async function apiCall(url, options) {
    const response = await fetch(url, options);
    const data = await response.json();
    if (!response.ok) throw new Error(data.message || 'An error occurred');
    return data;
}

export default Lobby;
