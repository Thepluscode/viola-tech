import type { Metadata } from "next";
import "./globals.css";
import { Sidebar } from "@/components/layout/sidebar";

export const metadata: Metadata = {
  title: {
    default: "Viola XDR",
    template: "%s | Viola XDR",
  },
  description: "Viola Extended Detection and Response — SOC Console",
  icons: {
    icon: [
      { url: "/favicon.ico", sizes: "32x32" },
      { url: "/favicon.svg", type: "image/svg+xml" },
    ],
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className="dark">
      <body className="bg-viola-bg text-viola-text antialiased">
        <div className="flex min-h-screen">
          <Sidebar />
          <main className="flex-1 ml-56 min-h-screen overflow-x-hidden">
            {children}
          </main>
        </div>
      </body>
    </html>
  );
}
