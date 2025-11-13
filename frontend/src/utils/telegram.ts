// Утилиты для работы с Telegram Mini App

declare global {
  interface Window {
    Telegram?: {
      WebApp: {
        initData: string;
        initDataUnsafe: {
          user?: {
            id: number;
            first_name?: string;
            last_name?: string;
            username?: string;
            language_code?: string;
          };
          auth_date?: number;
          hash?: string;
        };
        ready: () => void;
        expand: () => void;
        close: () => void;
        version: string;
        platform: string;
      };
    };
  }
}

/**
 * Получает init_data из Telegram WebApp
 */
export function getInitData(): string | null {
  if (typeof window === 'undefined') {
    return null;
  }

  const tg = window.Telegram?.WebApp;
  if (!tg) {
    return null;
  }

  return tg.initData || null;
}

/**
 * Проверяет, запущено ли приложение в Telegram
 */
export function isTelegramWebApp(): boolean {
  if (typeof window === 'undefined') {
    return false;
  }

  return !!window.Telegram?.WebApp;
}

/**
 * Инициализирует Telegram WebApp
 */
export function initTelegramWebApp(): void {
  if (typeof window === 'undefined') {
    return;
  }

  const tg = window.Telegram?.WebApp;
  if (!tg) {
    return;
  }

  tg.ready();
  tg.expand();
}

/**
 * Создаёт заголовки для API запросов с init_data
 */
export function getApiHeaders(): HeadersInit {
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
  };

  const initData = getInitData();
  if (initData) {
    headers['init_data'] = initData;
  }

  return headers;
}

/**
 * Обёртка для fetch с автоматической передачей init_data
 */
export async function apiFetch(
  url: string,
  options: RequestInit = {}
): Promise<Response> {
  const headers = new Headers(options.headers);
  const apiHeaders = getApiHeaders();

  // Добавляем init_data в заголовки
  Object.entries(apiHeaders).forEach(([key, value]) => {
    headers.set(key, value);
  });

  return fetch(url, {
    ...options,
    headers,
  });
}

