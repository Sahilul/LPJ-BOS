<?php

declare(strict_types=1);

namespace App;

/** Kesalahan isian: pesan per field, dipakai bersama oleh form web dan tool MCP. */
final class ValidationException extends \RuntimeException
{
    public function __construct(public readonly array $errors, string $message = 'Isian belum valid.')
    {
        parent::__construct($message);
    }
}
