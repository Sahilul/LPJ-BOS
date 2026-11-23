<?php

declare(strict_types=1);

namespace App;

final class Router
{
    private array $routes = [];

    public function get(string $pattern, array|callable $handler): void
    {
        $this->routes[] = ['GET', $pattern, $handler];
    }

    public function post(string $pattern, array|callable $handler): void
    {
        $this->routes[] = ['POST', $pattern, $handler];
    }

    public function dispatch(string $method, string $uri): void
    {
        $path = '/' . trim(parse_url($uri, PHP_URL_PATH) ?: '/', '/');
        $allowed = false;

        foreach ($this->routes as [$m, $pattern, $handler]) {
            $regex = '#^' . preg_replace('#\{(\w+)\}#', '(?P<$1>[^/]+)', rtrim($pattern, '/') ?: '/') . '$#';
            if (!preg_match($regex, $path, $match)) {
                continue;
            }
            if ($m !== $method) {
                $allowed = true;
                continue;
            }
            $params = array_map('rawurldecode', array_filter($match, 'is_string', ARRAY_FILTER_USE_KEY));
            if (is_array($handler)) {
                [$class, $action] = $handler;
                (new $class())->$action(...$params);
            } else {
                $handler(...$params);
            }
            return;
        }

        abort($allowed ? 405 : 404, $allowed ? 'Metode tidak diizinkan.' : 'Halaman tidak ditemukan.');
    }
}
