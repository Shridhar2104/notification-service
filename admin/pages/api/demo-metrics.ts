import type { NextApiRequest, NextApiResponse } from 'next';

const metrics = [
  { label: 'Messages (24h)', value: '128,420', sublabel: '+12% vs yesterday' },
  { label: 'Delivery Rate', value: '98.7%', sublabel: 'p95 latency 62ms' },
  { label: 'Active Tenants', value: '84', sublabel: '5 enterprise' }
];

export default function handler(_: NextApiRequest, res: NextApiResponse) {
  res.status(200).json(metrics);
}
