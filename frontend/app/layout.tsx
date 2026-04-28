import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Cancer To Hell ",
  description: "Local oncology co-pilot for tumor board decision support",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="h-full antialiased">
      <body className="min-h-full bg-zinc-50 text-zinc-900">{children}</body>
    </html>
  );
}
