import type { Metadata } from "next";
import "./globals.css";
export const metadata: Metadata = {
  title: "Knitwork | Shared tools. More possibilities.",
  description:
    "Equipment, workspaces and skilled support for local garment makers.",
};
export default function Layout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
