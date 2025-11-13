import { useState, useEffect } from 'react';
import { ListingCard, type ListingCardData } from './ListingCard';
import { Tabs, TabsList, TabsTrigger } from './ui/tabs';
import { Button } from './ui/button';
import { FilterScroll } from './FilterScroll';
import { apiFetch } from '../utils/telegram';

type MainCategory = 'services' | 'buysell' | 'other';

export function ListingsTab() {
  const [mainCategory, setMainCategory] = useState<MainCategory>('services');
  const [serviceFilter, setServiceFilter] = useState<string>('offer');
  const [serviceType, setServiceType] = useState<string>('all');
  const [buysellFilter, setBuysellFilter] = useState<string>('sell');
  const [buysellType, setBuysellType] = useState<string>('all');
  const [otherType, setOtherType] = useState<string>('all');
  const [listings, setListings] = useState<ListingCardData[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchListings();
  }, [mainCategory, serviceFilter, serviceType, buysellFilter, buysellType, otherType]);

  const fetchListings = async () => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams();
      params.set('cat', mainCategory);

      if (mainCategory === 'services' && serviceType !== 'all') {
        params.set('tag', serviceType);
      } else if (mainCategory === 'buysell' && buysellType !== 'all') {
        params.set('tag', buysellType);
      } else if (mainCategory === 'other' && otherType !== 'all') {
        params.set('tag', otherType);
      }

      if (mainCategory === 'services') {
        params.set('mode', serviceFilter);
      }
      if (mainCategory === 'buysell') {
        params.set('mode', buysellFilter);
      }

      const response = await apiFetch(`/api/ads?${params.toString()}`);
      if (!response.ok) {
        throw new Error('Ошибка загрузки объявлений');
      }
      const data = await response.json();

      const transformedListings: ListingCardData[] = data.map((ad: any) => ({
        id: ad.id,
        title: ad.title,
        description: ad.desc,
        username: `@${ad.username}`,
        isPremium: ad.is_premium,
        category: ad.category,
        mode: ad.mode,
        tag: ad.tag,
        status: ad.status,
        expiresAt: ad.expires_at,
        photoUrl: ad.photo_url ?? null,
      }));
      setListings(transformedListings);
    } catch (error) {
      console.error('Failed to fetch listings:', error);
      setError('Не удалось загрузить объявления. Попробуйте обновить позже.');
      setListings([]);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="pb-4">
      {/* Header */}
      <div className="p-4 pb-0">
        <h1 className="text-2xl mb-1 pt-2">Объявления</h1>
        <p className="text-muted-foreground text-sm">Найдите услуги и предложения</p>
      </div>

      {/* Main Category Tabs */}
      <div className="px-4 pt-4">
        <Tabs value={mainCategory} onValueChange={(v) => setMainCategory(v as MainCategory)}>
          <TabsList className="w-full bg-muted p-1 rounded-xl">
            <TabsTrigger
              value="services"
              className="flex-1 rounded-lg data-[state=active]:bg-background data-[state=active]:text-[#FF0000]"
            >
              Услуги
            </TabsTrigger>
            <TabsTrigger
              value="buysell"
              className="flex-1 rounded-lg data-[state=active]:bg-background data-[state=active]:text-[#FF0000]"
            >
              Купля/Продажа
            </TabsTrigger>
            <TabsTrigger
              value="other"
              className="flex-1 rounded-lg data-[state=active]:bg-background data-[state=active]:text-[#FF0000]"
            >
              Другое
            </TabsTrigger>
          </TabsList>
        </Tabs>
      </div>

      {/* Filters */}
      <div className="px-4 pt-4 space-y-3">
        {mainCategory === 'services' && (
          <>
            <div className="flex gap-2">
              <Button
                onClick={() => {
                  setServiceFilter('offer');
                  setServiceType('all');
                }}
                variant={serviceFilter === 'offer' ? 'default' : 'outline'}
                className={`rounded-full whitespace-nowrap ${
                  serviceFilter === 'offer'
                    ? 'bg-[#FF0000] hover:bg-[#CC0000] text-white'
                    : 'border-border hover:border-[#FF0000]'
                }`}
              >
                Предлагаю услугу
              </Button>
              <Button
                onClick={() => {
                  setServiceFilter('search');
                  setServiceType('all');
                }}
                variant={serviceFilter === 'search' ? 'default' : 'outline'}
                className={`rounded-full whitespace-nowrap ${
                  serviceFilter === 'search'
                    ? 'bg-[#FF0000] hover:bg-[#CC0000] text-white'
                    : 'border-border hover:border-[#FF0000]'
                }`}
              >
                Ищу услугу
              </Button>
            </div>
            <FilterScroll>
              <Button
                onClick={() => setServiceType('all')}
                variant={serviceType === 'all' ? 'default' : 'outline'}
                size="sm"
                className={`rounded-full ${
                  serviceType === 'all'
                    ? 'bg-[#FF0000] hover:bg-[#CC0000] text-white'
                    : 'border-border'
                }`}
              >
                Все
              </Button>
              <Button
                onClick={() => setServiceType('designer')}
                variant={serviceType === 'designer' ? 'default' : 'outline'}
                size="sm"
                className={`rounded-full ${
                  serviceType === 'designer'
                    ? 'bg-[#FF0000] hover:bg-[#CC0000] text-white'
                    : 'border-border'
                }`}
              >
                Дизайнер
              </Button>
              <Button
                onClick={() => setServiceType('script')}
                variant={serviceType === 'script' ? 'default' : 'outline'}
                size="sm"
                className={`rounded-full ${
                  serviceType === 'script'
                    ? 'bg-[#FF0000] hover:bg-[#CC0000] text-white'
                    : 'border-border'
                }`}
              >
                Сценарист
              </Button>
              <Button
                onClick={() => setServiceType('voice')}
                variant={serviceType === 'voice' ? 'default' : 'outline'}
                size="sm"
                className={`rounded-full ${
                  serviceType === 'voice'
                    ? 'bg-[#FF0000] hover:bg-[#CC0000] text-white'
                    : 'border-border'
                }`}
              >
                Озвучивание
              </Button>
              <Button
                onClick={() => setServiceType('other')}
                variant={serviceType === 'other' ? 'default' : 'outline'}
                size="sm"
                className={`rounded-full ${
                  serviceType === 'other'
                    ? 'bg-[#FF0000] hover:bg-[#CC0000] text-white'
                    : 'border-border'
                }`}
              >
                Другое
              </Button>
            </FilterScroll>
          </>
        )}

        {mainCategory === 'buysell' && (
          <>
            <div className="flex gap-2">
              <Button
                onClick={() => {
                  setBuysellFilter('sell');
                  setBuysellType('all');
                }}
                variant={buysellFilter === 'sell' ? 'default' : 'outline'}
                className={`rounded-full whitespace-nowrap ${
                  buysellFilter === 'sell'
                    ? 'bg-[#FF0000] hover:bg-[#CC0000] text-white'
                    : 'border-border hover:border-[#FF0000]'
                }`}
              >
                Продам
              </Button>
              <Button
                onClick={() => {
                  setBuysellFilter('buy');
                  setBuysellType('all');
                }}
                variant={buysellFilter === 'buy' ? 'default' : 'outline'}
                className={`rounded-full whitespace-nowrap ${
                  buysellFilter === 'buy'
                    ? 'bg-[#FF0000] hover:bg-[#CC0000] text-white'
                    : 'border-border hover:border-[#FF0000]'
                }`}
              >
                Куплю
              </Button>
            </div>
            <FilterScroll>
              <Button
                onClick={() => setBuysellType('all')}
                variant={buysellType === 'all' ? 'default' : 'outline'}
                size="sm"
                className={`rounded-full ${
                  buysellType === 'all'
                    ? 'bg-[#FF0000] hover:bg-[#CC0000] text-white'
                    : 'border-border'
                }`}
              >
                Все
              </Button>
              <Button
                onClick={() => setBuysellType('konechka')}
                variant={buysellType === 'konechka' ? 'default' : 'outline'}
                size="sm"
                className={`rounded-full ${
                  buysellType === 'konechka'
                    ? 'bg-[#FF0000] hover:bg-[#CC0000] text-white'
                    : 'border-border'
                }`}
              >
                Конечка
              </Button>
              <Button
                onClick={() => setBuysellType('channel')}
                variant={buysellType === 'channel' ? 'default' : 'outline'}
                size="sm"
                className={`rounded-full ${
                  buysellType === 'channel'
                    ? 'bg-[#FF0000] hover:bg-[#CC0000] text-white'
                    : 'border-border'
                }`}
              >
                Канал
              </Button>
              <Button
                onClick={() => setBuysellType('video')}
                variant={buysellType === 'video' ? 'default' : 'outline'}
                size="sm"
                className={`rounded-full ${
                  buysellType === 'video'
                    ? 'bg-[#FF0000] hover:bg-[#CC0000] text-white'
                    : 'border-border'
                }`}
              >
                Видео
              </Button>
              <Button
                onClick={() => setBuysellType('adsense')}
                variant={buysellType === 'adsense' ? 'default' : 'outline'}
                size="sm"
                className={`rounded-full ${
                  buysellType === 'adsense'
                    ? 'bg-[#FF0000] hover:bg-[#CC0000] text-white'
                    : 'border-border'
                }`}
              >
                Адсенс
              </Button>
              <Button
                onClick={() => setBuysellType('templates')}
                variant={buysellType === 'templates' ? 'default' : 'outline'}
                size="sm"
                className={`rounded-full ${
                  buysellType === 'templates'
                    ? 'bg-[#FF0000] hover:bg-[#CC0000] text-white'
                    : 'border-border'
                }`}
              >
                Шаблоны
              </Button>
            </FilterScroll>
          </>
        )}

        {mainCategory === 'other' && (
          <FilterScroll>
            <Button
              onClick={() => setOtherType('all')}
              variant={otherType === 'all' ? 'default' : 'outline'}
              size="sm"
              className={`rounded-full ${
                otherType === 'all'
                  ? 'bg-[#FF0000] hover:bg-[#CC0000] text-white'
                  : 'border-border'
              }`}
            >
              Все
            </Button>
            <Button
              onClick={() => setOtherType('education')}
              variant={otherType === 'education' ? 'default' : 'outline'}
              size="sm"
              className={`rounded-full ${
                otherType === 'education'
                  ? 'bg-[#FF0000] hover:bg-[#CC0000] text-white'
                  : 'border-border'
              }`}
            >
              Обучение
            </Button>
            <Button
              onClick={() => setOtherType('courses')}
              variant={otherType === 'courses' ? 'default' : 'outline'}
              size="sm"
              className={`rounded-full ${
                otherType === 'courses'
                  ? 'bg-[#FF0000] hover:bg-[#CC0000] text-white'
                  : 'border-border'
              }`}
            >
              Курсы
            </Button>
            <Button
              onClick={() => setOtherType('cheats')}
              variant={otherType === 'cheats' ? 'default' : 'outline'}
              size="sm"
              className={`rounded-full ${
                otherType === 'cheats'
                  ? 'bg-[#FF0000] hover:bg-[#CC0000] text-white'
                  : 'border-border'
              }`}
            >
              Читы
            </Button>
            <Button
              onClick={() => setOtherType('mods')}
              variant={otherType === 'mods' ? 'default' : 'outline'}
              size="sm"
              className={`rounded-full ${
                otherType === 'mods'
                  ? 'bg-[#FF0000] hover:bg-[#CC0000] text-white'
                  : 'border-border'
              }`}
            >
              Моды
            </Button>
            <Button
              onClick={() => setOtherType('niche')}
              variant={otherType === 'niche' ? 'default' : 'outline'}
              size="sm"
              className={`rounded-full ${
                otherType === 'niche'
                  ? 'bg-[#FF0000] hover:bg-[#CC0000] text-white'
                  : 'border-border'
              }`}
            >
              Ниша
            </Button>
            <Button
              onClick={() => setOtherType('schemes')}
              variant={otherType === 'schemes' ? 'default' : 'outline'}
              size="sm"
              className={`rounded-full ${
                otherType === 'schemes'
                  ? 'bg-[#FF0000] hover:bg-[#CC0000] text-white'
                  : 'border-border'
              }`}
            >
              Схемы
            </Button>
            <Button
              onClick={() => setOtherType('boost')}
              variant={otherType === 'boost' ? 'default' : 'outline'}
              size="sm"
              className={`rounded-full ${
                otherType === 'boost'
                  ? 'bg-[#FF0000] hover:bg-[#CC0000] text-white'
                  : 'border-border'
              }`}
            >
              Накрутка
            </Button>
          </FilterScroll>
        )}
      </div>

      {/* Listings Grid */}
      <div className="px-4 pt-4 space-y-4">
        {loading ? (
          <div className="text-center py-12 text-muted-foreground">
            <p>Загрузка объявлений...</p>
          </div>
        ) : error ? (
          <div className="text-center py-12 text-destructive">
            <p>{error}</p>
          </div>
        ) : listings.length === 0 ? (
          <div className="text-center py-12 text-muted-foreground">
            <p>Объявления не найдены</p>
          </div>
        ) : (
          listings.map((listing) => (
            <ListingCard key={listing.id} listing={listing} />
          ))
        )}
      </div>
    </div>
  );
}
