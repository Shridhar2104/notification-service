import useSWR from 'swr';

const fetcher = (path: string) => fetch(path).then(res => res.json());

type Metric = {
  label: string;
  value: string;
  sublabel: string;
};

const demoMetrics: Metric[] = [
  { label: 'Messages (24h)', value: '128,420', sublabel: '+12% vs yesterday' },
  { label: 'Delivery Rate', value: '98.7%', sublabel: 'p95 latency 62ms' },
  { label: 'Active Tenants', value: '84', sublabel: '5 enterprise' },
];

export default function Dashboard() {
  const { data } = useSWR('/api/demo-metrics', fetcher, { fallbackData: demoMetrics });

  return (
    <div>
      <h2>Delivery Snapshot</h2>
      <div className="card-grid">
        {(data ?? demoMetrics).map(metric => (
          <div className="card" key={metric.label}>
            <h3>{metric.label}</h3>
            <p style={{ fontSize: '2rem', margin: '0.5rem 0' }}>{metric.value}</p>
            <span className="badge">{metric.sublabel}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
