import React, { useState, useEffect } from 'react';
import './Leaderboard.css';

const API_URL = process.env.REACT_APP_API_URL || (process.env.NODE_ENV === 'production'
  ? `https://${window.location.host}/api`
  : 'http://localhost:8080/api');

const Leaderboard = ({ isVisible, onClose, refreshTrigger }) => {
  const [leaderboard, setLeaderboard] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [lastUpdated, setLastUpdated] = useState(null);

  // Fetch leaderboard data
  const fetchLeaderboard = async () => {
    setLoading(true);
    setError(null);
    
    try {
      const response = await fetch(`${API_URL}/leaderboard`);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }
      
      const data = await response.json();
      setLeaderboard(data || []);
      setLastUpdated(new Date());
    } catch (err) {
      console.error('Failed to fetch leaderboard:', err);
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  // Initial fetch and refresh on trigger
  useEffect(() => {
    if (isVisible) {
      fetchLeaderboard();
    }
  }, [isVisible, refreshTrigger]);

  // Auto-refresh every 30 seconds when visible
  useEffect(() => {
    if (!isVisible) return;

    const interval = setInterval(fetchLeaderboard, 30000);
    return () => clearInterval(interval);
  }, [isVisible]);

  if (!isVisible) return null;

  return (
    <div className="leaderboard-overlay">
      <div className="leaderboard-modal">
        <div className="leaderboard-header">
          <h2>🏆 Leaderboard</h2>
          <div className="leaderboard-controls">
            <button 
              className="refresh-button"
              onClick={fetchLeaderboard}
              disabled={loading}
              title="Refresh leaderboard"
            >
              {loading ? '⏳' : '🔄'}
            </button>
            <button 
              className="close-button"
              onClick={onClose}
              title="Close leaderboard"
            >
              ✕
            </button>
          </div>
        </div>
        
        {lastUpdated && (
          <div className="last-updated">
            Last updated: {lastUpdated.toLocaleTimeString()}
          </div>
        )}
        
        <div className="leaderboard-content">
          {loading && leaderboard.length === 0 ? (
            <div className="loading-state">
              <div className="loading-spinner">⏳</div>
              <p>Loading leaderboard...</p>
            </div>
          ) : error ? (
            <div className="error-state">
              <div className="error-icon">❌</div>
              <p>Failed to load leaderboard</p>
              <p className="error-details">{error}</p>
              <button onClick={fetchLeaderboard} className="retry-button">
                Try Again
              </button>
            </div>
          ) : leaderboard.length === 0 ? (
            <div className="empty-state">
              <div className="empty-icon">🎮</div>
              <p>No games played yet!</p>
              <p className="empty-subtitle">Play some games to see the leaderboard</p>
            </div>
          ) : (
            <div className="leaderboard-table-container">
              <table className="leaderboard-table">
                <thead>
                  <tr>
                    <th>Rank</th>
                    <th>Player</th>
                    <th>Wins</th>
                    <th>Losses</th>
                    <th>Draws</th>
                    <th>Win Rate</th>
                  </tr>
                </thead>
                <tbody>
                  {leaderboard.map((player, index) => (
                    <tr key={player.player_name} className="leaderboard-row">
                      <td className="rank-cell">
                        <span className={`rank-badge rank-${getRankClass(index)}`}>
                          {getRankDisplay(index)}
                        </span>
                      </td>
                      <td className="player-cell">
                        <span className="player-name">{player.player_name}</span>
                      </td>
                      <td className="wins-cell">
                        <span className="stat-value">{player.wins}</span>
                      </td>
                      <td className="losses-cell">
                        <span className="stat-value">{player.losses}</span>
                      </td>
                      <td className="draws-cell">
                        <span className="stat-value">{player.draws}</span>
                      </td>
                      <td className="winrate-cell">
                        <div className="winrate-container">
                          <span className="winrate-value">{player.win_rate.toFixed(1)}%</span>
                          <div className="winrate-bar">
                            <div 
                              className="winrate-fill"
                              style={{ width: `${Math.min(player.win_rate, 100)}%` }}
                            />
                          </div>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
        
        <div className="leaderboard-footer">
          <p className="leaderboard-note">
            * Only players with 5+ games are shown
          </p>
          <p className="leaderboard-stats">
            Showing {leaderboard.length} players
          </p>
        </div>
      </div>
    </div>
  );
};

// Helper functions
const getRankClass = (index) => {
  if (index === 0) return 'gold';
  if (index === 1) return 'silver';
  if (index === 2) return 'bronze';
  return 'default';
};

const getRankDisplay = (index) => {
  if (index === 0) return '🥇';
  if (index === 1) return '🥈';
  if (index === 2) return '🥉';
  return index + 1;
};

export default Leaderboard;