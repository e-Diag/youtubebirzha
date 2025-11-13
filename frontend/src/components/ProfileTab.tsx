import { useState, useEffect } from 'react';
import { ListingCard, MANAGER_LINK, type ListingCardData } from './ListingCard';
import { Button } from './ui/button';
import { User, Moon, Sun } from 'lucide-react';
import { apiFetch } from '../utils/telegram';

interface ProfileTabProps {
  isDark: boolean;
  toggleTheme: () => void;
}

export function ProfileTab({ isDark, toggleTheme }: ProfileTabProps) {
  const [listings, setListings] = useState<ListingCardData[]>([]);
  const [loading, setLoading] = useState(true);
  const [userId] = useState<string | null>(() => {
    // Try to get user_id from Telegram WebApp first
    const tg = (window as any).Telegram?.WebApp;
    console.log('ProfileTab: проверка Telegram WebApp', { 
      hasTelegram: !!(window as any).Telegram, 
      hasWebApp: !!tg,
      hasUser: !!tg?.initDataUnsafe?.user,
      userId: tg?.initDataUnsafe?.user?.id 
    });
    if (tg?.initDataUnsafe?.user?.id) {
      const id = String(tg.initDataUnsafe.user.id);
      console.log('ProfileTab: userId получен из Telegram WebApp:', id);
      return id;
    }
    // Fallback to URL params or localStorage
    const params = new URLSearchParams(window.location.search);
    const urlUserId = params.get('user_id');
    const localUserId = localStorage.getItem('user_id');
    console.log('ProfileTab: userId из URL или localStorage', { urlUserId, localUserId });
    return urlUserId || localUserId;
  });

  useEffect(() => {
    if (userId) {
      fetchMyAds();
    } else {
      setLoading(false);
    }
  }, [userId]);

  const fetchMyAds = async () => {
    if (!userId) {
      console.log('ProfileTab: userId не найден, пропускаем запрос');
      return;
    }
    
    console.log('ProfileTab: запрос объявлений для user_id=', userId);
    setLoading(true);
    try {
      const response = await apiFetch(`/api/myads?user_id=${userId}`);
      console.log('ProfileTab: получен ответ', response.status, response.statusText);
      const data = await response.json();
      console.log('ProfileTab: получено объявлений', data.length);
      
      const transformedListings: ListingCardData[] = data.map((ad: any) => ({
        id: ad.id,
        title: ad.title,
        description: ad.desc,
        username: `@${ad.username}`,
        isPremium: ad.is_premium,
        category: ad.category,
        mode: ad.mode,
        tag: ad.tag,
        status: (ad.status ?? 'active') as 'active' | 'expired' | 'inactive',
        expiresAt: ad.expires_at,
        photoUrl: ad.photo_url ?? null,
      }));
      
      setListings(transformedListings);
    } catch (error) {
      console.error('Failed to fetch my ads:', error);
      setListings([]);
    } finally {
      setLoading(false);
    }
  };

  const hasListings = listings.length > 0;

  return (
    <div className="p-4 space-y-6">
      {/* Header with Theme Toggle */}
      <div className="pt-2">
        <div className="flex items-center justify-between mb-2">
          <h1 className="text-2xl">Профиль</h1>
          <Button
            onClick={toggleTheme}
            variant="outline"
            size="icon"
            className="rounded-xl border-border"
          >
            {isDark ? <Sun size={20} /> : <Moon size={20} />}
          </Button>
        </div>
        <p className="text-muted-foreground">Управление вашими объявлениями</p>
      </div>

      {/* No Listings State */}
      {!loading && !hasListings && (
        <div className="flex flex-col items-center justify-center py-16 space-y-6">
          <div className="w-24 h-24 bg-muted rounded-full flex items-center justify-center">
            <User size={48} className="text-muted-foreground" />
          </div>
          
          <div className="text-center space-y-2">
            <p className="text-muted-foreground">У вас нет объявлений</p>
            <p className="text-muted-foreground/60 text-sm">
              Свяжитесь с менеджером, чтобы разместить своё первое объявление
            </p>
          </div>

          <Button
            asChild
            className="bg-[#FF0000] hover:bg-[#CC0000] text-white px-8 py-6 rounded-2xl shadow-lg"
          >
            <a href={MANAGER_LINK} target="_blank" rel="noopener noreferrer">
              Обратитесь к менеджеру
            </a>
          </Button>
        </div>
      )}

      {/* Listings State */}
      {loading ? (
        <div className="text-center py-12 text-muted-foreground">
          <p>Загрузка объявлений...</p>
        </div>
      ) : hasListings ? (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <p className="text-muted-foreground">Мои объявления</p>
            <Button
              asChild
              className="bg-[#FF0000] hover:bg-[#CC0000] text-white rounded-xl"
              size="sm"
            >
              <a href={MANAGER_LINK} target="_blank" rel="noopener noreferrer">
                Обратиться к менеджеру
              </a>
            </Button>
          </div>
          
          {listings.map((listing) => {
            const isExpired = listing.status === 'expired';
            const isInactive = listing.status === 'inactive';

            return (
              <ListingCard
                key={listing.id}
                listing={listing}
                showExpiryDate={true}
                footer={
                  (isExpired || isInactive) && (
                    <div className="flex flex-col gap-2">
                      <p className="text-sm text-muted-foreground">
                        Объявление не показывается на бирже. Свяжитесь с менеджером, чтобы поднять его вновь.
                      </p>
                      <Button
                        asChild
                        className="bg-[#FF0000] hover:bg-[#CC0000] text-white"
                      >
                        <a href={MANAGER_LINK} target="_blank" rel="noopener noreferrer">
                          Обратитесь к менеджеру
                        </a>
                      </Button>
                    </div>
                  )
                }
              />
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
