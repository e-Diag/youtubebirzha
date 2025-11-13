import { useEffect, useState } from 'react';
import { Search, Shield } from 'lucide-react';
import { Input } from './ui/input';
import { Button } from './ui/button';
import { ScrollArea } from './ui/scroll-area';
import { apiFetch } from '../utils/telegram';

interface BlacklistEntry {
  username: string;
  created_at: string;
  updated_at: string;
}

export function BlacklistTab() {
  const [username, setUsername] = useState('');
  const [searchResult, setSearchResult] = useState<'scammer' | 'clean' | null>(null);
  const [entries, setEntries] = useState<BlacklistEntry[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [listLoading, setListLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [listError, setListError] = useState<string | null>(null);

  useEffect(() => {
    loadBlacklist();
  }, []);

  const loadBlacklist = async () => {
    setListLoading(true);
    setListError(null);
    try {
      const response = await apiFetch('/api/blacklist');
      if (!response.ok) {
        throw new Error('Ошибка загрузки чёрного списка');
      }
      const data = await response.json();
      setEntries(data);
    } catch (err) {
      console.error(err);
      setListError('Не удалось загрузить чёрный список. Попробуйте позже.');
    } finally {
      setListLoading(false);
    }
  };

  const handleSearch = async () => {
    if (!username.trim()) return;
    setLoading(true);
    setError(null);
    setSearchResult(null);
    
    try {
      const cleanUsername = username.trim().replace('@', '');
      const response = await apiFetch(`/api/scammer/${cleanUsername}`);
      if (!response.ok) {
        throw new Error('Ошибка проверки пользователя');
      }
      const data = await response.json();
      
      if (data.safe === false) {
        setSearchResult('scammer');
      } else {
        setSearchResult('clean');
      }
    } catch (error) {
      console.error('Failed to check user:', error);
      setError('Не удалось проверить пользователя. Попробуйте ещё раз.');
    }
    setLoading(false);
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !loading) {
      handleSearch();
    }
  };

  return (
    <div className="p-4 space-y-6 pb-24">
      {/* Header */}
      <div className="pt-2">
        <h1 className="text-2xl mb-2">Чёрный список</h1>
        <p className="text-muted-foreground">Проверьте пользователя перед сделкой</p>
      </div>

      {/* Search Field */}
      <div className="space-y-3">
        <div className="relative">
          <Input
            type="text"
            placeholder="@username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            onKeyPress={handleKeyPress}
            className="pr-12 h-12 rounded-xl border-border focus:border-[#FF0000] focus:ring-[#FF0000]"
          />
          <Button
            onClick={() => !loading && handleSearch()}
            disabled={loading}
            className="absolute right-1 top-1 h-10 w-10 p-0 bg-[#FF0000] hover:bg-[#CC0000] rounded-lg"
          >
            <Search size={20} />
          </Button>
        </div>
        {error && <p className="text-sm text-destructive">{error}</p>}
      </div>

      {/* Search Result */}
      {loading ? (
        <div className="text-center py-8 text-muted-foreground text-sm">Идёт проверка...</div>
      ) : searchResult && (
        <div className="space-y-4 animate-in fade-in duration-300">
          <div
            className={`p-6 rounded-2xl shadow-md border-2 ${
              searchResult === 'scammer'
                ? 'border-red-500 bg-red-50'
                : 'border-green-500 bg-green-50'
            }`}
          >
            <p
              className={`text-center ${
                searchResult === 'scammer' ? 'text-red-700' : 'text-green-700'
              }`}
            >
              {searchResult === 'scammer'
                ? '⚠️ Осторожно! Мошенник'
                : '✅ Юзер не был замечен в мошеннических схемах'}
            </p>
          </div>

          <p className="text-center text-muted-foreground text-sm">
            Если ошибка — обратитесь к{' '}
            <a
              href="https://t.me/birzha_manager"
              target="_blank"
              rel="noopener noreferrer"
              className="text-[#FF0000] hover:underline font-medium"
            >
              менеджеру
            </a>
          </p>
        </div>
      )}

      {/* Empty State */}
      {!searchResult && (
        <div className="text-center py-12 text-muted-foreground">
          <Shield size={64} className="mx-auto mb-4 opacity-30" />
          <p>Введите username для проверки</p>
        </div>
      )}

      {/* Blacklist Table */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">Все отмеченные пользователи</h2>
          <Button variant="ghost" size="sm" onClick={loadBlacklist} disabled={listLoading}>
            Обновить
          </Button>
        </div>
        {listError && <p className="text-sm text-destructive">{listError}</p>}
        <div className="rounded-2xl border border-border">
          <ScrollArea className="max-h-64">
            {listLoading ? (
              <div className="p-6 text-center text-muted-foreground text-sm">Загрузка списка...</div>
            ) : entries.length === 0 ? (
              <div className="p-6 text-center text-muted-foreground text-sm">
                Чёрный список пуст.
              </div>
            ) : (
              <div className="divide-y divide-border">
                {entries.map((entry) => (
                  <div key={entry.username} className="px-4 py-3 flex items-center justify-between">
                    <span className="font-medium">@{entry.username}</span>
                    <span className="text-xs text-muted-foreground">
                      обновлён {new Date(entry.updated_at).toLocaleDateString('ru-RU')}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </ScrollArea>
        </div>
      </div>
    </div>
  );
}
