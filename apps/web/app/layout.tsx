import type { Metadata } from "next";
import { RootFrame } from "@/components/RootFrame";
import { themeBootstrapScript } from "@/lib/theme";
import "./globals.css";

export const metadata: Metadata = {
  title: "RentStage Admin",
  description: "Rental operations, inventory and booking platform",
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="es" suppressHydrationWarning>
      <head><script dangerouslySetInnerHTML={{ __html: themeBootstrapScript() }} /></head>
      <body><RootFrame>{children}</RootFrame></body>
    </html>
  );
}
