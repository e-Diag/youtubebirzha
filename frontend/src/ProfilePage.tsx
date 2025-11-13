// В ProfilePage.tsx
const username = webApp.initDataUnsafe.user?.username;
const res = await fetch(`/api/profile/${username}`);
const ads = await res.json();