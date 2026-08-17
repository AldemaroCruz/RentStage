import type { Metadata } from "next";
import { RootFrame } from "@/components/RootFrame";
import "./globals.css";

export const metadata: Metadata = {
  title: "RentStage Admin",
  description: "Rental operations, inventory and booking platform",
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="es">
      <body><RootFrame>{children}</RootFrame></body>
    </html>
  );
}
