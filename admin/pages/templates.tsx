const templates = [
  { name: 'Welcome Email', channel: 'Email', version: 'v5', updated: '2 days ago' },
  { name: 'OTP SMS', channel: 'SMS', version: 'v2', updated: '4 hours ago' },
  { name: 'Re-engagement Push', channel: 'Push', version: 'v1', updated: '1 week ago' }
];

export default function Templates() {
  return (
    <div>
      <h2>Templates</h2>
      <table className="table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Channel</th>
            <th>Version</th>
            <th>Last Updated</th>
          </tr>
        </thead>
        <tbody>
          {templates.map(template => (
            <tr key={template.name}>
              <td>{template.name}</td>
              <td>{template.channel}</td>
              <td>{template.version}</td>
              <td>{template.updated}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
