import type { Metadata } from 'next'
import './globals.css'
import { Space_Grotesk, Inter, JetBrains_Mono } from 'next/font/google'

// AURA UI REBUILD
const space = Space_Grotesk({ subsets: ['latin'], variable: '--font-space' })
const inter = Inter({ subsets: ['latin'], variable: '--font-inter' })
const mono = JetBrains_Mono({ subsets: ['latin'], variable: '--font-mono' })

export const metadata: Metadata = {
  title: 'AURA — The Trust Protocol',
  description: 'Trusted Intelligence for Autonomous Systems',
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${space.variable} ${inter.variable} ${mono.variable}`}>
      <body className="min-h-dvh antialiased">
        {children}
      </body>
    </html>
  )
}
