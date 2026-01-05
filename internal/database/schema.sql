-- Connect Four Database Schema
-- PostgreSQL version 12+

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Games table - stores completed game records
CREATE TABLE IF NOT EXISTS games (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Player information
    player1_id UUID NOT NULL,
    player1_name VARCHAR(255) NOT NULL,
    player1_is_bot BOOLEAN DEFAULT FALSE,
    
    player2_id UUID NOT NULL,
    player2_name VARCHAR(255) NOT NULL,
    player2_is_bot BOOLEAN DEFAULT FALSE,
    
    -- Game outcome
    winner_id UUID, -- NULL for draws
    winner_name VARCHAR(255),
    is_draw BOOLEAN DEFAULT FALSE,
    
    -- Game details
    total_moves INTEGER NOT NULL DEFAULT 0,
    duration_seconds INTEGER NOT NULL DEFAULT 0,
    win_type VARCHAR(50), -- 'horizontal', 'vertical', 'diagonal_positive', 'diagonal_negative', 'forfeit', NULL for draw
    
    -- Game board final state (JSON)
    final_board JSONB,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    finished_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT games_winner_check CHECK (
        (winner_id IS NULL AND is_draw = TRUE) OR 
        (winner_id IS NOT NULL AND is_draw = FALSE)
    ),
    CONSTRAINT games_duration_positive CHECK (duration_seconds >= 0),
    CONSTRAINT games_moves_positive CHECK (total_moves >= 0),
    CONSTRAINT games_different_players CHECK (player1_id != player2_id)
);

-- Leaderboard table - aggregated player statistics
CREATE TABLE IF NOT EXISTS leaderboard (
    id SERIAL PRIMARY KEY,
    
    -- Player information
    username VARCHAR(255) UNIQUE NOT NULL,
    player_id UUID, -- Optional: link to a players table if you have one
    
    -- Game statistics
    total_games INTEGER DEFAULT 0,
    wins INTEGER DEFAULT 0,
    losses INTEGER DEFAULT 0,
    draws INTEGER DEFAULT 0,
    
    -- Performance metrics
    win_rate DECIMAL(5,2) DEFAULT 0.00, -- Percentage with 2 decimal places
    average_game_duration DECIMAL(8,2) DEFAULT 0.00, -- Average seconds
    total_playtime_seconds BIGINT DEFAULT 0,
    
    -- Win type breakdown
    horizontal_wins INTEGER DEFAULT 0,
    vertical_wins INTEGER DEFAULT 0,
    diagonal_wins INTEGER DEFAULT 0,
    forfeit_wins INTEGER DEFAULT 0,
    
    -- Opponent statistics
    wins_vs_humans INTEGER DEFAULT 0,
    wins_vs_bots INTEGER DEFAULT 0,
    losses_vs_humans INTEGER DEFAULT 0,
    losses_vs_bots INTEGER DEFAULT 0,
    
    -- Streaks and achievements
    current_win_streak INTEGER DEFAULT 0,
    longest_win_streak INTEGER DEFAULT 0,
    current_loss_streak INTEGER DEFAULT 0,
    
    -- Timestamps
    first_game_at TIMESTAMP WITH TIME ZONE,
    last_game_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT leaderboard_stats_consistent CHECK (
        total_games = wins + losses + draws
    ),
    CONSTRAINT leaderboard_win_rate_valid CHECK (
        win_rate >= 0.00 AND win_rate <= 100.00
    ),
    CONSTRAINT leaderboard_positive_stats CHECK (
        total_games >= 0 AND wins >= 0 AND losses >= 0 AND draws >= 0
    )
);

-- Game moves table - detailed move history (optional, for analytics)
CREATE TABLE IF NOT EXISTS game_moves (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    
    -- Move details
    player_id UUID NOT NULL,
    player_name VARCHAR(255) NOT NULL,
    player_number INTEGER NOT NULL CHECK (player_number IN (1, 2)),
    
    move_number INTEGER NOT NULL,
    column_played INTEGER NOT NULL CHECK (column_played >= 0 AND column_played <= 6),
    row_landed INTEGER NOT NULL CHECK (row_landed >= 0 AND row_landed <= 5),
    
    -- Move context
    board_state_before JSONB,
    board_state_after JSONB,
    
    -- Bot-specific data
    is_bot_move BOOLEAN DEFAULT FALSE,
    bot_reasoning TEXT,
    bot_confidence INTEGER CHECK (bot_confidence >= 0 AND bot_confidence <= 100),
    
    -- Timing
    move_timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    time_taken_ms INTEGER DEFAULT 0,
    
    UNIQUE(game_id, move_number)
);

-- Indexes for performance

-- Games table indexes
CREATE INDEX IF NOT EXISTS idx_games_player1_id ON games(player1_id);
CREATE INDEX IF NOT EXISTS idx_games_player2_id ON games(player2_id);
CREATE INDEX IF NOT EXISTS idx_games_winner_id ON games(winner_id);
CREATE INDEX IF NOT EXISTS idx_games_finished_at ON games(finished_at);
CREATE INDEX IF NOT EXISTS idx_games_duration ON games(duration_seconds);
CREATE INDEX IF NOT EXISTS idx_games_created_at ON games(created_at);
CREATE INDEX IF NOT EXISTS idx_games_player_names ON games(player1_name, player2_name);

-- Leaderboard table indexes
CREATE INDEX IF NOT EXISTS idx_leaderboard_username ON leaderboard(username);
CREATE INDEX IF NOT EXISTS idx_leaderboard_wins ON leaderboard(wins DESC);
CREATE INDEX IF NOT EXISTS idx_leaderboard_win_rate ON leaderboard(win_rate DESC);
CREATE INDEX IF NOT EXISTS idx_leaderboard_total_games ON leaderboard(total_games DESC);
CREATE INDEX IF NOT EXISTS idx_leaderboard_last_game ON leaderboard(last_game_at DESC);

-- Game moves table indexes
CREATE INDEX IF NOT EXISTS idx_game_moves_game_id ON game_moves(game_id);
CREATE INDEX IF NOT EXISTS idx_game_moves_player_id ON game_moves(player_id);
CREATE INDEX IF NOT EXISTS idx_game_moves_timestamp ON game_moves(move_timestamp);

-- Composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_games_player_outcome ON games(player1_name, winner_name, finished_at);
CREATE INDEX IF NOT EXISTS idx_leaderboard_ranking ON leaderboard(win_rate DESC, wins DESC, total_games DESC);

-- Views for common queries

-- Player game history view
CREATE OR REPLACE VIEW player_game_history AS
SELECT 
    g.id as game_id,
    g.finished_at,
    g.duration_seconds,
    g.total_moves,
    g.win_type,
    g.is_draw,
    
    -- Player perspective
    CASE 
        WHEN p.player_name = g.player1_name THEN 1
        WHEN p.player_name = g.player2_name THEN 2
    END as player_number,
    
    p.player_name,
    p.player_id,
    p.is_bot as player_is_bot,
    
    -- Opponent info
    CASE 
        WHEN p.player_name = g.player1_name THEN g.player2_name
        WHEN p.player_name = g.player2_name THEN g.player1_name
    END as opponent_name,
    
    CASE 
        WHEN p.player_name = g.player1_name THEN g.player2_is_bot
        WHEN p.player_name = g.player2_name THEN g.player1_is_bot
    END as opponent_is_bot,
    
    -- Game outcome from player perspective
    CASE 
        WHEN g.is_draw THEN 'draw'
        WHEN (p.player_name = g.player1_name AND g.winner_name = g.player1_name) OR
             (p.player_name = g.player2_name AND g.winner_name = g.player2_name) THEN 'win'
        ELSE 'loss'
    END as outcome
    
FROM games g
CROSS JOIN (
    SELECT DISTINCT player1_name as player_name, player1_id as player_id, player1_is_bot as is_bot FROM games
    UNION
    SELECT DISTINCT player2_name as player_name, player2_id as player_id, player2_is_bot as is_bot FROM games
) p
WHERE p.player_name IN (g.player1_name, g.player2_name);

-- Leaderboard ranking view
CREATE OR REPLACE VIEW leaderboard_ranking AS
SELECT 
    ROW_NUMBER() OVER (ORDER BY win_rate DESC, wins DESC, total_games DESC) as rank,
    username,
    total_games,
    wins,
    losses,
    draws,
    win_rate,
    CASE 
        WHEN total_games >= 10 THEN 'active'
        WHEN total_games >= 5 THEN 'casual'
        ELSE 'beginner'
    END as player_tier,
    last_game_at,
    current_win_streak,
    longest_win_streak
FROM leaderboard
WHERE total_games > 0
ORDER BY rank;

-- Functions for maintaining data integrity

-- Function to update leaderboard after game completion
CREATE OR REPLACE FUNCTION update_leaderboard_stats()
RETURNS TRIGGER AS $$
BEGIN
    -- Update player1 stats
    INSERT INTO leaderboard (username, player_id, total_games, wins, losses, draws, first_game_at, last_game_at)
    VALUES (
        NEW.player1_name, 
        NEW.player1_id, 
        1,
        CASE WHEN NEW.winner_name = NEW.player1_name THEN 1 ELSE 0 END,
        CASE WHEN NEW.winner_name != NEW.player1_name AND NOT NEW.is_draw THEN 1 ELSE 0 END,
        CASE WHEN NEW.is_draw THEN 1 ELSE 0 END,
        NEW.finished_at,
        NEW.finished_at
    )
    ON CONFLICT (username) DO UPDATE SET
        total_games = leaderboard.total_games + 1,
        wins = leaderboard.wins + CASE WHEN NEW.winner_name = NEW.player1_name THEN 1 ELSE 0 END,
        losses = leaderboard.losses + CASE WHEN NEW.winner_name != NEW.player1_name AND NOT NEW.is_draw THEN 1 ELSE 0 END,
        draws = leaderboard.draws + CASE WHEN NEW.is_draw THEN 1 ELSE 0 END,
        win_rate = CASE 
            WHEN (leaderboard.total_games + 1) > 0 
            THEN ROUND((leaderboard.wins + CASE WHEN NEW.winner_name = NEW.player1_name THEN 1 ELSE 0 END) * 100.0 / (leaderboard.total_games + 1), 2)
            ELSE 0.00 
        END,
        total_playtime_seconds = leaderboard.total_playtime_seconds + NEW.duration_seconds,
        average_game_duration = ROUND((leaderboard.total_playtime_seconds + NEW.duration_seconds) / (leaderboard.total_games + 1.0), 2),
        last_game_at = NEW.finished_at,
        updated_at = NOW(),
        
        -- Update win type counters
        horizontal_wins = leaderboard.horizontal_wins + CASE WHEN NEW.winner_name = NEW.player1_name AND NEW.win_type = 'horizontal' THEN 1 ELSE 0 END,
        vertical_wins = leaderboard.vertical_wins + CASE WHEN NEW.winner_name = NEW.player1_name AND NEW.win_type = 'vertical' THEN 1 ELSE 0 END,
        diagonal_wins = leaderboard.diagonal_wins + CASE WHEN NEW.winner_name = NEW.player1_name AND NEW.win_type LIKE 'diagonal%' THEN 1 ELSE 0 END,
        forfeit_wins = leaderboard.forfeit_wins + CASE WHEN NEW.winner_name = NEW.player1_name AND NEW.win_type = 'forfeit' THEN 1 ELSE 0 END,
        
        -- Update opponent type counters
        wins_vs_humans = leaderboard.wins_vs_humans + CASE WHEN NEW.winner_name = NEW.player1_name AND NOT NEW.player2_is_bot THEN 1 ELSE 0 END,
        wins_vs_bots = leaderboard.wins_vs_bots + CASE WHEN NEW.winner_name = NEW.player1_name AND NEW.player2_is_bot THEN 1 ELSE 0 END,
        losses_vs_humans = leaderboard.losses_vs_humans + CASE WHEN NEW.winner_name != NEW.player1_name AND NOT NEW.is_draw AND NOT NEW.player2_is_bot THEN 1 ELSE 0 END,
        losses_vs_bots = leaderboard.losses_vs_bots + CASE WHEN NEW.winner_name != NEW.player1_name AND NOT NEW.is_draw AND NEW.player2_is_bot THEN 1 ELSE 0 END;

    -- Update player2 stats
    INSERT INTO leaderboard (username, player_id, total_games, wins, losses, draws, first_game_at, last_game_at)
    VALUES (
        NEW.player2_name, 
        NEW.player2_id, 
        1,
        CASE WHEN NEW.winner_name = NEW.player2_name THEN 1 ELSE 0 END,
        CASE WHEN NEW.winner_name != NEW.player2_name AND NOT NEW.is_draw THEN 1 ELSE 0 END,
        CASE WHEN NEW.is_draw THEN 1 ELSE 0 END,
        NEW.finished_at,
        NEW.finished_at
    )
    ON CONFLICT (username) DO UPDATE SET
        total_games = leaderboard.total_games + 1,
        wins = leaderboard.wins + CASE WHEN NEW.winner_name = NEW.player2_name THEN 1 ELSE 0 END,
        losses = leaderboard.losses + CASE WHEN NEW.winner_name != NEW.player2_name AND NOT NEW.is_draw THEN 1 ELSE 0 END,
        draws = leaderboard.draws + CASE WHEN NEW.is_draw THEN 1 ELSE 0 END,
        win_rate = CASE 
            WHEN (leaderboard.total_games + 1) > 0 
            THEN ROUND((leaderboard.wins + CASE WHEN NEW.winner_name = NEW.player2_name THEN 1 ELSE 0 END) * 100.0 / (leaderboard.total_games + 1), 2)
            ELSE 0.00 
        END,
        total_playtime_seconds = leaderboard.total_playtime_seconds + NEW.duration_seconds,
        average_game_duration = ROUND((leaderboard.total_playtime_seconds + NEW.duration_seconds) / (leaderboard.total_games + 1.0), 2),
        last_game_at = NEW.finished_at,
        updated_at = NOW(),
        
        -- Update win type counters
        horizontal_wins = leaderboard.horizontal_wins + CASE WHEN NEW.winner_name = NEW.player2_name AND NEW.win_type = 'horizontal' THEN 1 ELSE 0 END,
        vertical_wins = leaderboard.vertical_wins + CASE WHEN NEW.winner_name = NEW.player2_name AND NEW.win_type = 'vertical' THEN 1 ELSE 0 END,
        diagonal_wins = leaderboard.diagonal_wins + CASE WHEN NEW.winner_name = NEW.player2_name AND NEW.win_type LIKE 'diagonal%' THEN 1 ELSE 0 END,
        forfeit_wins = leaderboard.forfeit_wins + CASE WHEN NEW.winner_name = NEW.player2_name AND NEW.win_type = 'forfeit' THEN 1 ELSE 0 END,
        
        -- Update opponent type counters
        wins_vs_humans = leaderboard.wins_vs_humans + CASE WHEN NEW.winner_name = NEW.player2_name AND NOT NEW.player1_is_bot THEN 1 ELSE 0 END,
        wins_vs_bots = leaderboard.wins_vs_bots + CASE WHEN NEW.winner_name = NEW.player2_name AND NEW.player1_is_bot THEN 1 ELSE 0 END,
        losses_vs_humans = leaderboard.losses_vs_humans + CASE WHEN NEW.winner_name != NEW.player2_name AND NOT NEW.is_draw AND NOT NEW.player1_is_bot THEN 1 ELSE 0 END,
        losses_vs_bots = leaderboard.losses_vs_bots + CASE WHEN NEW.winner_name != NEW.player2_name AND NOT NEW.is_draw AND NEW.player1_is_bot THEN 1 ELSE 0 END;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically update leaderboard
CREATE TRIGGER trigger_update_leaderboard
    AFTER INSERT ON games
    FOR EACH ROW
    EXECUTE FUNCTION update_leaderboard_stats();

-- Function to recalculate all leaderboard stats (for maintenance)
CREATE OR REPLACE FUNCTION recalculate_leaderboard_stats()
RETURNS void AS $$
BEGIN
    -- Clear existing leaderboard
    DELETE FROM leaderboard;
    
    -- Recalculate from games table
    INSERT INTO leaderboard (
        username, player_id, total_games, wins, losses, draws, 
        win_rate, total_playtime_seconds, average_game_duration,
        horizontal_wins, vertical_wins, diagonal_wins, forfeit_wins,
        wins_vs_humans, wins_vs_bots, losses_vs_humans, losses_vs_bots,
        first_game_at, last_game_at
    )
    SELECT 
        player_name,
        player_id,
        COUNT(*) as total_games,
        SUM(CASE WHEN outcome = 'win' THEN 1 ELSE 0 END) as wins,
        SUM(CASE WHEN outcome = 'loss' THEN 1 ELSE 0 END) as losses,
        SUM(CASE WHEN outcome = 'draw' THEN 1 ELSE 0 END) as draws,
        CASE 
            WHEN COUNT(*) > 0 
            THEN ROUND(SUM(CASE WHEN outcome = 'win' THEN 1 ELSE 0 END) * 100.0 / COUNT(*), 2)
            ELSE 0.00 
        END as win_rate,
        SUM(duration_seconds) as total_playtime_seconds,
        ROUND(AVG(duration_seconds), 2) as average_game_duration,
        SUM(CASE WHEN outcome = 'win' AND win_type = 'horizontal' THEN 1 ELSE 0 END) as horizontal_wins,
        SUM(CASE WHEN outcome = 'win' AND win_type = 'vertical' THEN 1 ELSE 0 END) as vertical_wins,
        SUM(CASE WHEN outcome = 'win' AND win_type LIKE 'diagonal%' THEN 1 ELSE 0 END) as diagonal_wins,
        SUM(CASE WHEN outcome = 'win' AND win_type = 'forfeit' THEN 1 ELSE 0 END) as forfeit_wins,
        SUM(CASE WHEN outcome = 'win' AND NOT opponent_is_bot THEN 1 ELSE 0 END) as wins_vs_humans,
        SUM(CASE WHEN outcome = 'win' AND opponent_is_bot THEN 1 ELSE 0 END) as wins_vs_bots,
        SUM(CASE WHEN outcome = 'loss' AND NOT opponent_is_bot THEN 1 ELSE 0 END) as losses_vs_humans,
        SUM(CASE WHEN outcome = 'loss' AND opponent_is_bot THEN 1 ELSE 0 END) as losses_vs_bots,
        MIN(finished_at) as first_game_at,
        MAX(finished_at) as last_game_at
    FROM player_game_history
    GROUP BY player_name, player_id;
END;
$$ LANGUAGE plpgsql;