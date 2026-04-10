import './App.css';
import { Dashboard } from './components/Dashboard';
import { TopologyView } from './components/TopologyView';

function App() {
  return (
    <div className="layout">
      <header>
        <h1>LAN Mapper</h1>
        <p>单设备部署的局域网拓扑洞察</p>
      </header>
      <main>
        <Dashboard />
        <TopologyView />
      </main>
    </div>
  );
}

export default App;
