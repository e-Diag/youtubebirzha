import { useState, useEffect, useRef, type ReactNode } from 'react';
import { Flame, Clock, ChevronDown, ChevronUp } from 'lucide-react';
import { ImageWithFallback } from './figma/ImageWithFallback';
import { Button } from './ui/button';

const MANAGER_LINK = 'https://t.me/birzha_manager';

const categoryLabels: Record<string, string> = {
  services: 'Услуги',
  buysell: 'Купля / Продажа',
  other: 'Другое',
};

const modeLabels: Record<string, string> = {
  offer: 'Предлагаю',
  search: 'Ищу',
  sell: 'Продаю',
  buy: 'Покупаю',
  general: 'Объявление',
};

const tagLabels: Record<string, string> = {
  all: 'Все',
  designer: 'Дизайн',
  script: 'Сценарий',
  voice: 'Озвучивание',
  other: 'Другое',
  konechka: 'Конечка',
  channel: 'Канал',
  video: 'Видео',
  adsense: 'AdSense',
  templates: 'Шаблоны',
  education: 'Обучение',
  courses: 'Курсы',
  cheats: 'Читы',
  mods: 'Моды',
  niche: 'Ниша',
  schemes: 'Схемы',
  boost: 'Накрутка',
};

export interface ListingCardData {
  id: number;
  title: string;
  description: string;
  username: string;
  isPremium: boolean;
  category?: string;
  mode?: string;
  tag?: string;
  status?: 'active' | 'expired' | 'inactive';
  expiresAt?: string;
  photoUrl?: string | null;
}

interface ListingCardProps {
  listing: ListingCardData;
  footer?: ReactNode;
  showExpiryDate?: boolean; // Показывать дату окончания только для владельца
  showFullDescription?: boolean; // Показывать полное описание без обрезки (для профиля)
}

const MAX_DESCRIPTION_LENGTH = 150; // Примерная длина для 3 строк

export function ListingCard({ listing, footer, showExpiryDate = false, showFullDescription = false }: ListingCardProps) {
  const [isExpanded, setIsExpanded] = useState(false); // По умолчанию всегда свернуто
  const [shouldShowExpand, setShouldShowExpand] = useState(false);
  const descriptionRef = useRef<HTMLParagraphElement>(null);
  const cardRef = useRef<HTMLDivElement>(null);

  const isExpired = listing.status === 'expired';
  const isInactive = listing.status === 'inactive';
  const isPremium = listing.isPremium;
  const hasPhoto = listing.photoUrl && listing.photoUrl.trim() !== '';

  // Проверяем, нужно ли показывать кнопку разворачивания
  // Проверка должна происходить после рендеринга, когда применён line-clamp
  useEffect(() => {
    const checkIfExpanded = () => {
      if (descriptionRef.current) {
        const element = descriptionRef.current;
        
        // Ждём, пока стили применятся
        requestAnimationFrame(() => {
          requestAnimationFrame(() => {
            if (element) {
              // Временно убираем line-clamp для проверки полной высоты
              const originalClass = element.className;
              const originalStyle = element.style.cssText;
              
              // Проверяем полную высоту без ограничений
              element.className = element.className.replace('line-clamp-3', '').trim();
              element.style.maxHeight = 'none';
              
              const fullHeight = element.scrollHeight;
              const lineHeight = parseFloat(getComputedStyle(element).lineHeight) || 20;
              const minHeight = lineHeight * 3;
              
              // Восстанавливаем оригинальные стили
              element.className = originalClass;
              element.style.cssText = originalStyle;
              
              // Показываем кнопку, если текст больше 3 строк
              setShouldShowExpand(fullHeight > minHeight);
            }
          });
        });
      }
    };

    // Проверяем с задержками, чтобы дать время на рендеринг и применение стилей
    const timeoutId1 = setTimeout(checkIfExpanded, 100);
    const timeoutId2 = setTimeout(checkIfExpanded, 300);
    const timeoutId3 = setTimeout(checkIfExpanded, 500);

    // Используем ResizeObserver для отслеживания изменений размера
    let resizeObserver: ResizeObserver | null = null;
    if (descriptionRef.current) {
      resizeObserver = new ResizeObserver(() => {
        // Добавляем небольшую задержку для ResizeObserver, чтобы стили успели примениться
        setTimeout(checkIfExpanded, 50);
      });
      resizeObserver.observe(descriptionRef.current);
    }

    return () => {
      clearTimeout(timeoutId1);
      clearTimeout(timeoutId2);
      clearTimeout(timeoutId3);
      if (resizeObserver) {
        resizeObserver.disconnect();
      }
    };
  }, [listing.description]); // Убрали isExpanded из зависимостей, чтобы кнопка не исчезала при клике

  // Автосворачивание при прокрутке
  useEffect(() => {
    if (!cardRef.current) return;

    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          // Если карточка выходит из видимой области, сворачиваем текст
          if (!entry.isIntersecting && isExpanded) {
            setIsExpanded(false);
          }
        });
      },
      {
        threshold: 0, // Когда карточка полностью выходит из видимой области
        rootMargin: '50px', // Небольшой отступ для более плавного сворачивания
      }
    );

    observer.observe(cardRef.current);

    return () => {
      observer.disconnect();
    };
  }, [isExpanded]);

  const borderClass = isExpired || isInactive
    ? 'border-2 border-red-500'
    : isPremium
      ? 'border-2 border-[#FF0000]'
      : 'border border-border';

  const statusLabel = isExpired ? 'Срок истёк' : isInactive ? 'Снят' : null;
  const expiresAtLabel = listing.expiresAt
    ? new Date(listing.expiresAt).toLocaleDateString('ru-RU', {
        day: '2-digit',
        month: '2-digit',
        year: 'numeric',
      })
    : null;

  const categoryLabel = listing.category ? categoryLabels[listing.category] ?? listing.category : null;
  const modeLabel = listing.mode ? modeLabels[listing.mode] ?? listing.mode : null;
  const tagLabel = listing.tag && listing.tag !== 'all' ? tagLabels[listing.tag] ?? listing.tag : null;

  return (
    <div ref={cardRef} className={`bg-card rounded-2xl shadow-md overflow-hidden transition-all hover:shadow-lg ${borderClass}`}>
      {hasPhoto && (
        <div className="relative aspect-video overflow-hidden bg-muted">
          <ImageWithFallback src={listing.photoUrl!} alt={listing.title} className="w-full h-full object-cover" />
          {isPremium && !isExpired && !isInactive && (
            <div className="absolute top-3 right-3 bg-[#FF0000] text-white px-3 py-1 rounded-full flex items-center gap-1 shadow-lg">
              <Flame size={16} />
              <span className="text-sm">Лучшее</span>
            </div>
          )}
          {(isExpired || isInactive) && (
            <div className="absolute top-3 right-3 bg-red-600 text-white px-3 py-1 rounded-full shadow-lg text-sm">
              Нет на бирже
            </div>
          )}
        </div>
      )}

      {!hasPhoto && (isExpired || isInactive) && (
        <div className="p-4 pb-0">
          <div className="inline-block bg-red-600 text-white px-3 py-1 rounded-full shadow-lg text-sm">
            Нет на бирже
          </div>
        </div>
      )}

      {!hasPhoto && isPremium && !isExpired && !isInactive && (
        <div className="p-4 pb-0">
          <div className="inline-flex items-center gap-1 bg-[#FF0000] text-white px-3 py-1 rounded-full shadow-lg">
            <Flame size={16} />
            <span className="text-sm">Лучшее</span>
          </div>
        </div>
      )}

      <div className="p-4 space-y-3">
        <div className="flex flex-col gap-1">
          <h3 className="line-clamp-1 text-base font-semibold">{listing.title}</h3>
          <div className="relative">
            <p
              ref={descriptionRef}
              className="text-muted-foreground text-sm transition-all"
              style={isExpanded ? { 
                maxHeight: 'none',
                display: 'block',
                overflow: 'visible',
                WebkitLineClamp: 'unset',
                WebkitBoxOrient: 'unset'
              } : {
                display: '-webkit-box',
                WebkitLineClamp: 3,
                WebkitBoxOrient: 'vertical',
                overflow: 'hidden',
                maxHeight: '4.5em' // Примерно 3 строки
              }}
            >
              {listing.description}
            </p>
            {shouldShowExpand && (
              <Button
                variant="ghost"
                size="sm"
                className="absolute right-0 top-0 h-auto p-1 text-muted-foreground hover:text-foreground z-10"
                onClick={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  setIsExpanded(!isExpanded);
                }}
                onMouseDown={(e) => {
                  // Предотвращаем исчезновение кнопки при клике
                  e.preventDefault();
                }}
              >
                {isExpanded ? (
                  <ChevronUp size={16} />
                ) : (
                  <ChevronDown size={16} />
                )}
              </Button>
            )}
          </div>
        </div>

        <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
          {categoryLabel && <span className="rounded-full bg-muted px-2 py-1">{categoryLabel}</span>}
          {modeLabel && <span className="rounded-full bg-muted px-2 py-1">{modeLabel}</span>}
          {tagLabel && <span className="rounded-full bg-muted px-2 py-1">{tagLabel}</span>}
        </div>

        <div className="flex items-center justify-between text-xs text-muted-foreground">
          {statusLabel && <span className="text-red-500 font-medium">{statusLabel}</span>}
          {showExpiryDate && expiresAtLabel && (
            <span className="flex items-center gap-1">
              <Clock size={14} />
              до {expiresAtLabel}
            </span>
          )}
        </div>

        <a
          href={`https://t.me/${listing.username.replace('@', '')}`}
          target="_blank"
          rel="noopener noreferrer"
          className="text-[#FF0000] hover:underline inline-block font-medium"
        >
          {listing.username}
        </a>

        {footer && (
          <div className="pt-3 border-t border-border">
            {footer}
          </div>
        )}
      </div>
    </div>
  );
}

export { MANAGER_LINK };
