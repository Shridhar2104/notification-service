import type { AppProps } from 'next/app';
import Head from 'next/head';
import Link from 'next/link';
import '../styles.css';

export default function App({ Component, pageProps }: AppProps) {
  return (
    <>
      <Head>
        <title>HSNP Admin</title>
        <meta name="viewport" content="width=device-width, initial-scale=1" />
      </Head>
      <div className="layout">
        <aside className="sidebar">
          <h1>HSNP</h1>
          <nav>
            <Link href="/">Dashboard</Link>
            <Link href="/tenants">Tenants</Link>
            <Link href="/templates">Templates</Link>
          </nav>
        </aside>
        <main className="content">
          <Component {...pageProps} />
        </main>
      </div>
    </>
  );
}
