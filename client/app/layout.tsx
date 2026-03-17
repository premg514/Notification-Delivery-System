import type { Metadata } from "next";
import Link from "next/link";
import "./globals.css";

export const metadata: Metadata = {
  title: "Notification Console",
  description: "Operational dashboard for notification delivery",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body>
        <header className="site-brand">
          <div className="site-brand-inner">
            <Link href="/" className="site-brand-link">
              AGH
            </Link>
            <span className="site-brand-subtitle">Notification Console</span>
          </div>
        </header>
        {children}
      </body>
    </html>
  );
}
