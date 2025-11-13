import { createRoot } from "react-dom/client";
import App from "./App.tsx";
import "./index.css";
import { initTelegramWebApp } from "./utils/telegram";

// Инициализируем Telegram WebApp
initTelegramWebApp();

createRoot(document.getElementById("root")!).render(<App />);
  