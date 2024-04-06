import React, { useState, useEffect, useRef } from 'react';
import backgroundImageSrc from '../assets/images/background.png';

const Lobby = () => {
	const [users, setUsers] = useState([]); // Holds the list of users and their ready status
	const wsRef = useRef(null);
	const accessToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MTI2ODkyNDksImlkIjoiMCIsInVzZXJuYW1lIjoiYWRtaW4ifQ.QNlKN9gW1CN5tC9g79Iwo8-TcPRnNFS51CTwuNwxKL8';

	useEffect(() => {
		wsRef.current = new WebSocket(`ws://localhost:8000/ws/${accessToken}`);

		wsRef.current.onopen = () => {
			console.log("Connected to the lobby");
	};

    wsRef.current.onmessage = (event) => {
		const message = JSON.parse(event.data);
		switch (message.type) {
		  case 'lobbyState':
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

	
    return () => {
		// Clean up the WebSocket connection when the component unmounts
		if (wsRef.current) {
		  wsRef.current.close();
		}
	};
	}, []);

	const handleReadyClick = () => {
		sendReady();
	};

	// Send ready message when the current user marks themselves as ready
	const sendReady = () => {
		const message = JSON.stringify({ action: "ready", timestamp: Date.now() });
		wsRef.current.send(message);
		wsRef.current.onmessage = (event) => {
			const message = JSON.parse(event.data);
			switch (message.type) {
			  case 'playerReady':
				console.log("Player is ready");
				users.find(user => user.username === 'admin').ready = true;
				setUsers([...users]);
				break;
			  case 'playerAlreadyReady':
				console.log("Player is already ready");
				break;
			  default:
				console.log("Received message: ", message);
			}
		  }
	};
	
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
			<button onClick={handleReadyClick}>I'm Ready</button>
		</div>
	  );
	};
	
export default Lobby;