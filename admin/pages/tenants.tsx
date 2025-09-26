const tenants = [
  { name: 'Acme Corp', plan: 'Enterprise', sendsPerDay: '45k', status: 'Active' },
  { name: 'Northwind', plan: 'Growth', sendsPerDay: '12k', status: 'Active' },
  { name: 'Globex', plan: 'Starter', sendsPerDay: '1.2k', status: 'Trial' }
];

export default function Tenants() {
  return (
    <div>
      <h2>Tenants</h2>
      <table className="table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Plan</th>
            <th>Daily Sends</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          {tenants.map(tenant => (
            <tr key={tenant.name}>
              <td>{tenant.name}</td>
              <td>{tenant.plan}</td>
              <td>{tenant.sendsPerDay}</td>
              <td>{tenant.status}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
