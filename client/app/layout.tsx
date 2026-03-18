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
            <div className="flex flex-col gap-5 xl:flex-row xl:items-center xl:justify-between">
            <div className="flex items-center gap-4">
              <div className="flex h-14 w-14 items-center justify-center rounded-3xl bg-coral text-3xl font-black text-white shadow-soft">
                A
              </div>
              <div>
                <p className="font-display text-3xl uppercase tracking-tight text-ink">
                  AGH
                </p>
                <p className="text-sm text-slate-500">
                  Notification operations console
                </p>
              </div>
            </div>
          </div>
          </div>
        </header>
        {children}
      </body>
    </html>
  );
}
