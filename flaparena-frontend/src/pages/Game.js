import React, { useRef, useEffect } from 'react';
import backgroundImageSrc from '../assets/images/background.png';
import Bird from '../components/Bird';
import PipePair from '../components/Pipe';
import pipeImageSrc from '../assets/images/pipe.png';

const Game = () => {
    const canvasRef = useRef(null);
    const birdPositionRef = useRef({ x: 250, y: 250 });
    const birdVelocityRef = useRef(0);
    const pipesRef = useRef([]); // useRef for pipes to persist without causing re-renders
    const gravity = 0.5;
    const jumpStrength = -10;
    const gapBetweenPipes = 450;
    const gapBetweenTopBottomPipe = 250;
    const wsRef = useRef(null);

    wsRef.current = new WebSocket("ws://localhost:8000/ws/eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MTI0MDg5NzksImlkIjoiOSIsInVzZXJuYW1lIjoidGVzdCJ9.H9QpzfFJXpOuzlz7pT-pc75cezBl9xDHCtdIoiRyqOU");

    wsRef.current.onopen = () => {
        console.log("Connected");
        wsRef.current.send(JSON.stringify({ action: "ready", timestamp: Date.now() }));
    };
    
    wsRef.current.onmessage = event => console.log("Received message: " + event.data);
    
    wsRef.current.onerror = error => console.log("WebSocket Error: " + error);
    
    wsRef.current.onclose = event => console.log("Disconnected", event.code, event.reason);
    
    useEffect(() => {
        // Load pipe images inside useEffect to ensure they're loaded after component mounts
        const pipeImage = new Image();
        pipeImage.src = pipeImageSrc;

        const background = new Image();
        background.src = backgroundImageSrc;

        // Make sure images are loaded before using them
        const imageLoadPromises = [
            new Promise(resolve => { background.onload = resolve; }),
            new Promise(resolve => { pipeImage.onload = resolve; }),
            Bird.loadImage(),
        ];

        Promise.all(imageLoadPromises).then(() => {
            const canvas = canvasRef.current;
            const ctx = canvas.getContext('2d');
            canvas.width = window.innerWidth;
            canvas.height = window.innerHeight;

            const draw = () => {
            ctx.clearRect(0, 0, canvas.width, canvas.height);
            ctx.drawImage(background, 0, 0, canvas.width, canvas.height);

            pipesRef.current.forEach(pipePair => {
                pipePair.draw(ctx);
                pipePair.update();
                if (checkCollision(birdPositionRef.current, Bird.image, pipePair, pipeImage)) {
                    // wsRef.current.send(JSON.stringify({ action: "dead", timestamp: Date.now() }));
                }
            });

            // Bird logic
            const currentPosition = birdPositionRef.current;
            currentPosition.y += birdVelocityRef.current;
            birdVelocityRef.current += gravity;
            
            // Draw the bird
            ctx.drawImage(Bird.image, currentPosition.x, currentPosition.y);

            // Boundary checks for the bird
            if (currentPosition.y + Bird.image.height > canvas.height) {
                currentPosition.y = canvas.height - Bird.image.height;
                birdVelocityRef.current = 0;
            } else if (currentPosition.y < 0) {
                currentPosition.y = 0;
                birdVelocityRef.current = 0;
            }

            // Remove pipes that have gone off screen and add new pipes
            pipesRef.current = pipesRef.current.filter(pipe => pipe.x + pipeImage.width > 0);
            if (pipesRef.current.length === 0 || pipesRef.current[pipesRef.current.length - 1].x < canvas.width - gapBetweenPipes) {
                // Add a new pipe at the right edge of the canvas
                const gapTop = Math.random() * (canvas.height - 300) + 50; // Randomize the gap position
                pipesRef.current.push(new PipePair(canvas.width, gapTop, gapBetweenTopBottomPipe, pipeImage));
            }
            };

            const gameLoop = setInterval(draw, 1000 / 60);

            const handleKeyDown = (event) => {
            if (event.key === 'ArrowUp' || event.key === ' ') {
                birdVelocityRef.current = jumpStrength;
            }
            };

            window.addEventListener('keydown', handleKeyDown);

            return () => {
            if (wsRef.current) {
                wsRef.current.close();
                }
                
                clearInterval(gameLoop);
                window.removeEventListener('keydown', handleKeyDown);
            };
        });
    }, []);

    return <canvas ref={canvasRef} style={{ width: '100%', height: '100vh', display: 'block' }} />;
};

// Collision detection function
function checkCollision(birdPosition, birdImage, pipePair, pipeImage) {
    // Define the bird's bounding box
    const birdBoundingBox = {
        x: birdPosition.x,
        y: birdPosition.y,
        width: birdImage.width - 30, // Reduce the width of the bounding box slightly
        height: birdImage.height - 30 // Reduce the height of the bounding box slightly
    };

    
    // Define the bounding boxes for the top and bottom pipes
    const topPipeBoundingBox = {
        x: pipePair.x,
        y: 0,
        width: pipeImage.width - 10,
        height: pipePair.gapTop - 20
    };
    
    
    const bottomPipeBoundingBox = {
        x: pipePair.x,
        y: pipePair.gapTop + pipePair.gapSize,
        width: pipeImage.width - 10,
        height: window.innerHeight - (pipePair.gapTop + pipePair.gapSize) + 15
    };
    
    // Check for overlaps between the bird's bounding box and the pipe bounding boxes
    return rectOverlap(birdBoundingBox, topPipeBoundingBox) ||
            rectOverlap(birdBoundingBox, bottomPipeBoundingBox);
    }

    // Helper function to check if two rectangles overlap
    function rectOverlap(rectA, rectB) {
    return (
        rectA.x < rectB.x + rectB.width &&
        rectA.x + rectA.width > rectB.x &&
        rectA.y < rectB.y + rectB.height &&
        rectA.y + rectA.height > rectB.y
    );
}

export default Game;