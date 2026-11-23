<?php

declare(strict_types=1);

namespace App;

use PDO;
use PDOStatement;

final class Database
{
    private static ?PDO $pdo = null;

    public static function pdo(): PDO
    {
        if (self::$pdo === null) {
            $path = config('db_path');
            if (!is_dir(dirname($path))) {
                mkdir(dirname($path), 0775, true);
            }
            $pdo = new PDO('sqlite:' . $path, null, null, [
                PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
                PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
            ]);
            $pdo->exec('PRAGMA foreign_keys = ON');
            $pdo->exec('PRAGMA journal_mode = WAL');
            $pdo->exec('PRAGMA busy_timeout = 5000');
            self::$pdo = $pdo;
        }
        return self::$pdo;
    }

    public static function close(): void
    {
        self::$pdo = null;
    }

    public static function run(string $sql, array $params = []): PDOStatement
    {
        $stmt = self::pdo()->prepare($sql);
        $stmt->execute($params);
        return $stmt;
    }

    public static function all(string $sql, array $params = []): array
    {
        return self::run($sql, $params)->fetchAll();
    }

    public static function one(string $sql, array $params = []): array|false
    {
        return self::run($sql, $params)->fetch();
    }

    public static function value(string $sql, array $params = []): mixed
    {
        return self::run($sql, $params)->fetchColumn();
    }

    public static function insert(string $table, array $data): int
    {
        $cols = array_keys($data);
        $sql = sprintf(
            'INSERT INTO %s (%s) VALUES (%s)',
            $table,
            implode(', ', $cols),
            implode(', ', array_fill(0, count($cols), '?'))
        );
        self::run($sql, array_values($data));
        return (int) self::pdo()->lastInsertId();
    }

    public static function update(string $table, array $data, int $id): void
    {
        $set = implode(', ', array_map(fn ($c) => "$c = ?", array_keys($data)));
        self::run("UPDATE $table SET $set WHERE id = ?", [...array_values($data), $id]);
    }

    public static function transaction(callable $fn): mixed
    {
        $pdo = self::pdo();
        $pdo->beginTransaction();
        try {
            $result = $fn();
            $pdo->commit();
            return $result;
        } catch (\Throwable $e) {
            $pdo->rollBack();
            throw $e;
        }
    }

    public static function migrate(): void
    {
        $pdo = self::pdo();
        $pdo->exec('CREATE TABLE IF NOT EXISTS migrations (name TEXT PRIMARY KEY, applied_at TEXT NOT NULL)');
        $done = array_flip($pdo->query('SELECT name FROM migrations')->fetchAll(PDO::FETCH_COLUMN));
        $files = glob(config('migrations_dir') . '/*.sql') ?: [];
        sort($files);
        foreach ($files as $file) {
            $name = basename($file);
            if (isset($done[$name])) {
                continue;
            }
            self::transaction(function () use ($pdo, $file, $name) {
                $pdo->exec((string) file_get_contents($file));
                self::run('INSERT INTO migrations (name, applied_at) VALUES (?, ?)', [$name, date('c')]);
            });
        }
    }
}
