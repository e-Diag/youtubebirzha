import { useState, useEffect } from 'react';
import { Shield, List, User } from 'lucide-react';
import { BlacklistTab } from './components/BlacklistTab';
import { ListingsTab } from './components/ListingsTab';
import { ProfileTab } from './components/ProfileTab';
import { isTelegramWebApp } from './utils/telegram';

export default function App() {
  const [activeTab, setActiveTab] = useState<'blacklist' | 'listings' | 'profile'>('listings');
  const [isDark, setIsDark] = useState(false);

  useEffect(() => {
    // Load theme preference from localStorage
    const savedTheme = localStorage.getItem('theme');
    if (savedTheme === 'dark') {
      setIsDark(true);
      document.documentElement.classList.add('dark');
    }
  }, []);

  const toggleTheme = () => {
    setIsDark(!isDark);
    if (!isDark) {
      document.documentElement.classList.add('dark');
      localStorage.setItem('theme', 'dark');
    } else {
      document.documentElement.classList.remove('dark');
      localStorage.setItem('theme', 'light');
    }
  };

  return (
    <div className="min-h-screen bg-background flex flex-col">
      {/* Content Area */}
      <div className="flex-1 overflow-y-auto pb-20">
        {activeTab === 'blacklist' && <BlacklistTab />}
        {activeTab === 'listings' && <ListingsTab />}
        {activeTab === 'profile' && <ProfileTab isDark={isDark} toggleTheme={toggleTheme} />}
      </div>

      {/* Bottom Navigation */}
      <nav className="fixed bottom-0 left-0 right-0 bg-background border-t border-border px-4 py-2 shadow-lg">
        <div className="flex justify-around items-center max-w-md mx-auto">
          <button
            onClick={() => setActiveTab('blacklist')}
            className={`flex flex-col items-center gap-1 px-4 py-2 rounded-lg transition-colors ${
              activeTab === 'blacklist' ? 'text-[#FF0000]' : 'text-muted-foreground'
            }`}
          >
            <Shield size={24} />
            <span className="text-xs">Чёрный список</span>
          </button>
          
          <button
            onClick={() => setActiveTab('listings')}
            className={`flex flex-col items-center gap-1 px-4 py-2 rounded-lg transition-colors ${
              activeTab === 'listings' ? 'text-[#FF0000]' : 'text-muted-foreground'
            }`}
          >
            <List size={24} />
            <span className="text-xs">Объявления</span>
          </button>
          
          <button
            onClick={() => setActiveTab('profile')}
            className={`flex flex-col items-center gap-1 px-4 py-2 rounded-lg transition-colors ${
              activeTab === 'profile' ? 'text-[#FF0000]' : 'text-muted-foreground'
            }`}
          >
            <User size={24} />
            <span className="text-xs">Профиль</span>
          </button>
        </div>
      </nav>
    </div>
  );
}
