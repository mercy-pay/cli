export const metadata = { title: 'Boutique de test' };

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="fr">
      <body style={{ fontFamily: 'system-ui, sans-serif', maxWidth: 480, margin: '40px auto', padding: '0 16px' }}>{children}</body>
    </html>
  );
}
