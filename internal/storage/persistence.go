package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Paulo-Ribeiro-Log/hpa-watchdog/internal/models"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
)

// PersistenceConfig configuração de persistência
type PersistenceConfig struct {
	Enabled     bool          // Habilita persistência
	DBPath      string        // Caminho do banco SQLite
	MaxAge      time.Duration // Máximo tempo de retenção (default: 24h)
	BatchSize   int           // Tamanho do batch para insert (default: 100)
	AutoCleanup bool          // Limpeza automática de dados antigos
}

// DefaultPersistenceConfig retorna configuração padrão
func DefaultPersistenceConfig() *PersistenceConfig {
	homeDir, _ := os.UserHomeDir()
	dbPath := filepath.Join(homeDir, ".hpa-watchdog", "snapshots.db")

	return &PersistenceConfig{
		Enabled:     true,
		DBPath:      dbPath,
		MaxAge:      24 * time.Hour,
		BatchSize:   100,
		AutoCleanup: true,
	}
}

// Persistence gerencia persistência em SQLite
type Persistence struct {
	config *PersistenceConfig
	db     *sql.DB
}

// NewPersistence cria nova instância de persistência
func NewPersistence(config *PersistenceConfig) (*Persistence, error) {
	if config == nil {
		config = DefaultPersistenceConfig()
	}

	if !config.Enabled {
		log.Info().Msg("Persistence disabled")
		return &Persistence{config: config}, nil
	}

	// Cria diretório se não existir
	dir := filepath.Dir(config.DBPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	// Abre/cria banco SQLite
	db, err := sql.Open("sqlite3", config.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configura connection pool
	db.SetMaxOpenConns(1) // SQLite funciona melhor com 1 conexão
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	p := &Persistence{
		config: config,
		db:     db,
	}

	// Inicializa schema
	if err := p.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	log.Info().
		Str("db_path", config.DBPath).
		Dur("max_age", config.MaxAge).
		Msg("Persistence initialized")

	// Cleanup inicial
	if config.AutoCleanup {
		if err := p.Cleanup(); err != nil {
			log.Warn().Err(err).Msg("Initial cleanup failed")
		}
	}

	return p, nil
}

// initSchema cria tabelas se não existirem
func (p *Persistence) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS snapshots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		cluster TEXT NOT NULL,
		namespace TEXT NOT NULL,
		hpa_name TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		data TEXT NOT NULL,  -- JSON do HPASnapshot
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(cluster, namespace, hpa_name, timestamp)
	);

	CREATE INDEX IF NOT EXISTS idx_snapshots_lookup
		ON snapshots(cluster, namespace, hpa_name, timestamp DESC);

	CREATE INDEX IF NOT EXISTS idx_snapshots_cleanup
		ON snapshots(timestamp);

	CREATE TABLE IF NOT EXISTS metadata (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := p.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Salva versão do schema
	_, err = p.db.Exec(`
		INSERT OR REPLACE INTO metadata (key, value, updated_at)
		VALUES ('schema_version', '1', CURRENT_TIMESTAMP)
	`)

	return err
}

// SaveSnapshot salva um snapshot no banco
func (p *Persistence) SaveSnapshot(snapshot *models.HPASnapshot) error {
	if !p.config.Enabled || p.db == nil {
		return nil
	}

	// Serializa snapshot para JSON
	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	// Insert (ignora duplicatas)
	_, err = p.db.Exec(`
		INSERT OR IGNORE INTO snapshots (cluster, namespace, hpa_name, timestamp, data)
		VALUES (?, ?, ?, ?, ?)
	`,
		snapshot.Cluster,
		snapshot.Namespace,
		snapshot.Name,
		snapshot.Timestamp,
		string(data),
	)

	if err != nil {
		return fmt.Errorf("failed to save snapshot: %w", err)
	}

	return nil
}

// SaveSnapshots salva múltiplos snapshots em batch
func (p *Persistence) SaveSnapshots(snapshots []*models.HPASnapshot) error {
	if !p.config.Enabled || p.db == nil || len(snapshots) == 0 {
		return nil
	}

	tx, err := p.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO snapshots (cluster, namespace, hpa_name, timestamp, data)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, snapshot := range snapshots {
		data, err := json.Marshal(snapshot)
		if err != nil {
			log.Warn().
				Err(err).
				Str("hpa", snapshot.Name).
				Msg("Failed to marshal snapshot, skipping")
			continue
		}

		_, err = stmt.Exec(
			snapshot.Cluster,
			snapshot.Namespace,
			snapshot.Name,
			snapshot.Timestamp,
			string(data),
		)
		if err != nil {
			log.Warn().
				Err(err).
				Str("hpa", snapshot.Name).
				Msg("Failed to insert snapshot, skipping")
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Debug().
		Int("count", len(snapshots)).
		Msg("Snapshots saved to database")

	return nil
}

// LoadSnapshots carrega snapshots de um HPA específico
func (p *Persistence) LoadSnapshots(cluster, namespace, name string, since time.Time) ([]models.HPASnapshot, error) {
	if !p.config.Enabled || p.db == nil {
		return nil, nil
	}

	rows, err := p.db.Query(`
		SELECT data FROM snapshots
		WHERE cluster = ? AND namespace = ? AND hpa_name = ?
		  AND timestamp >= ?
		ORDER BY timestamp ASC
	`, cluster, namespace, name, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query snapshots: %w", err)
	}
	defer rows.Close()

	snapshots := make([]models.HPASnapshot, 0)
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			log.Warn().Err(err).Msg("Failed to scan snapshot")
			continue
		}

		var snapshot models.HPASnapshot
		if err := json.Unmarshal([]byte(data), &snapshot); err != nil {
			log.Warn().Err(err).Msg("Failed to unmarshal snapshot")
			continue
		}

		snapshots = append(snapshots, snapshot)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating snapshots: %w", err)
	}

	return snapshots, nil
}

// LoadAll carrega todos os snapshots recentes (últimos MaxAge)
func (p *Persistence) LoadAll(since time.Time) (map[string][]models.HPASnapshot, error) {
	if !p.config.Enabled || p.db == nil {
		return nil, nil
	}

	rows, err := p.db.Query(`
		SELECT cluster, namespace, hpa_name, data FROM snapshots
		WHERE timestamp >= ?
		ORDER BY cluster, namespace, hpa_name, timestamp ASC
	`, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query all snapshots: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]models.HPASnapshot)
	for rows.Next() {
		var cluster, namespace, name, data string
		if err := rows.Scan(&cluster, &namespace, &name, &data); err != nil {
			log.Warn().Err(err).Msg("Failed to scan snapshot")
			continue
		}

		var snapshot models.HPASnapshot
		if err := json.Unmarshal([]byte(data), &snapshot); err != nil {
			log.Warn().Err(err).Msg("Failed to unmarshal snapshot")
			continue
		}

		key := fmt.Sprintf("%s/%s/%s", cluster, namespace, name)
		result[key] = append(result[key], snapshot)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating snapshots: %w", err)
	}

	log.Info().
		Int("hpas", len(result)).
		Time("since", since).
		Msg("Loaded snapshots from database")

	return result, nil
}

// Cleanup remove snapshots antigos (> MaxAge)
func (p *Persistence) Cleanup() error {
	if !p.config.Enabled || p.db == nil {
		return nil
	}

	cutoff := time.Now().Add(-p.config.MaxAge)

	result, err := p.db.Exec(`
		DELETE FROM snapshots
		WHERE timestamp < ?
	`, cutoff)
	if err != nil {
		return fmt.Errorf("failed to cleanup snapshots: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows > 0 {
		log.Info().
			Int64("removed", rows).
			Time("cutoff", cutoff).
			Msg("Cleanup: removed old snapshots")
	}

	// VACUUM para reduzir tamanho do arquivo
	if rows > 1000 {
		if _, err := p.db.Exec("VACUUM"); err != nil {
			log.Warn().Err(err).Msg("Failed to vacuum database")
		}
	}

	return nil
}

// Stats retorna estatísticas do banco
func (p *Persistence) Stats() (*PersistenceStats, error) {
	if !p.config.Enabled || p.db == nil {
		return &PersistenceStats{Enabled: false}, nil
	}

	stats := &PersistenceStats{
		Enabled: true,
		DBPath:  p.config.DBPath,
	}

	// Total snapshots
	err := p.db.QueryRow(`SELECT COUNT(*) FROM snapshots`).Scan(&stats.TotalSnapshots)
	if err != nil {
		return nil, fmt.Errorf("failed to count snapshots: %w", err)
	}

	// Total HPAs
	err = p.db.QueryRow(`
		SELECT COUNT(DISTINCT cluster || '/' || namespace || '/' || hpa_name)
		FROM snapshots
	`).Scan(&stats.TotalHPAs)
	if err != nil {
		return nil, fmt.Errorf("failed to count HPAs: %w", err)
	}

	// Oldest/Newest - SQLite retorna timestamps como string
	var oldestStr, newestStr sql.NullString
	err = p.db.QueryRow(`SELECT MIN(timestamp), MAX(timestamp) FROM snapshots`).
		Scan(&oldestStr, &newestStr)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get timestamp range: %w", err)
	}

	if oldestStr.Valid {
		if t, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", oldestStr.String); err == nil {
			stats.OldestSnapshot = t
		}
	}
	if newestStr.Valid {
		if t, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", newestStr.String); err == nil {
			stats.NewestSnapshot = t
		}
	}

	// Tamanho do arquivo
	fileInfo, err := os.Stat(p.config.DBPath)
	if err == nil {
		stats.DBSize = fileInfo.Size()
	}

	return stats, nil
}

// PersistenceStats estatísticas de persistência
type PersistenceStats struct {
	Enabled        bool
	DBPath         string
	DBSize         int64
	TotalSnapshots int64
	TotalHPAs      int64
	OldestSnapshot time.Time
	NewestSnapshot time.Time
}

// Close fecha conexão com banco
func (p *Persistence) Close() error {
	if p.db != nil {
		log.Info().Msg("Closing database connection")
		return p.db.Close()
	}
	return nil
}

// Vacuum executa VACUUM no banco (compacta)
func (p *Persistence) Vacuum() error {
	if !p.config.Enabled || p.db == nil {
		return nil
	}

	log.Info().Msg("Running VACUUM on database")
	_, err := p.db.Exec("VACUUM")
	return err
}
