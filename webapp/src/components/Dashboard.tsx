import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { fetchDevices } from '../api/client';

export function Dashboard() {
  const devicesQuery = useQuery({ queryKey: ['devices'], queryFn: fetchDevices, refetchInterval: 30000 });
  const devices = useMemo(() => (Array.isArray(devicesQuery.data) ? devicesQuery.data : []), [devicesQuery.data]);
  const stats = useMemo(() => ({
    total: devices.length,
    switches: devices.filter((d) => d.type?.toLowerCase().includes('switch')).length,
    routers: devices.filter((d) => d.type?.toLowerCase().includes('router')).length,
  }), [devices]);

  if (devicesQuery.isLoading) {
    return <div className="dashboard state">正在扫描网络…</div>;
  }

  if (devicesQuery.isError) {
    return <div className="dashboard state error">获取设备数据失败</div>;
  }

  if (!devices.length) {
    return (
      <div className="dashboard empty">
        <h3>暂无设备</h3>
        <p>扫描完成后会自动刷新</p>
      </div>
    );
  }

  return (
    <div className="dashboard">
      <div>
        <h3>设备总数</h3>
        <p>{stats.total}</p>
      </div>
      <div>
        <h3>交换机</h3>
        <p>{stats.switches}</p>
      </div>
      <div>
        <h3>路由器</h3>
        <p>{stats.routers}</p>
      </div>
    </div>
  );
}
