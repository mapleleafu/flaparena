import React from 'react';
import { BrowserRouter as Router, Route, Routes, Navigate } from 'react-router-dom';
import Game from './pages/Game';
import Lobby from './pages/Lobby';

const App = () => {
  return (
    <Router>
      <Routes>
        <Route path="/game" element={<Game />} />
        <Route path="/lobby" element={<Lobby />} />
        <Route path="*" element={<Navigate replace to="/lobby" />} />
      </Routes>
    </Router>
  );
};

export default App;
