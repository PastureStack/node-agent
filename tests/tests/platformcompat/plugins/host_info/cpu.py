import platform
import os
import math

from tests.platformcompat.utils import CadvisorAPIClient
from tests.platformcompat import Config


class CpuCollector(object):
    def __init__(self):
        self.cadvisor = CadvisorAPIClient(Config.cadvisor_ip(),
                                          Config.cadvisor_port())

    def _get_cpuinfo_data(self):
        with open('/proc/cpuinfo') as f:
            data = f.readlines()

        return data

    def _get_linux_cpu_info(self):
        data = {}

        procs = []
        file_data = self._get_cpuinfo_data()
        for line in file_data:
            split_line = line.split(':')
            if split_line[0].strip() == "model name":
                procs.append(split_line[1].strip())
                frequency = ghz_to_mhz(split_line[1])
                if frequency is not None:
                    data['mhz'] = frequency

            if 'mhz' not in data:
                if split_line[0].strip() == "cpu MHz":
                    data['mhz'] = float(split_line[1].strip())

        data['modelName'] = procs[0]
        data['count'] = len(procs)

        return data

    def _get_cpu_percentages(self):
        data = {}
        data['cpuCoresPercentages'] = []

        stats = self.cadvisor.get_stats()

        if len(stats) >= 2:
            stat_latest = stats[-1]
            stat_prev = stats[-2]

            time_diff = self.cadvisor.timestamp_diff(stat_latest['timestamp'],
                                                     stat_prev['timestamp'])

            latest_usage = stat_latest['cpu']['usage']['per_cpu_usage']
            prev_usage = stat_prev['cpu']['usage']['per_cpu_usage']

            for idx, core_usage in enumerate(latest_usage):
                cpu_usage = float(core_usage) - float(prev_usage[idx])
                percentage = (cpu_usage / time_diff) * 100
                percentage = round(percentage, 3)

                if percentage > 100:
                    percentage = math.floor(percentage)

                data['cpuCoresPercentages'].append(percentage)

        return data

    def _get_load_average(self):
        return {'loadAvg': list(os.getloadavg())}

    def key_name(self):
        return "cpuInfo"

    def get_data(self):
        data = {}

        if platform.system() == 'Linux':
            data.update(self._get_linux_cpu_info())
            data.update(self._get_load_average())
            data.update(self._get_cpu_percentages())

        return data

    def get_labels(self, pfx="rancher"):
        if os.path.exists('/dev/kvm'):
            return {".".join([pfx, "kvm"]): "true"}
        else:
            return {}


def ghz_to_mhz(model_name):
    for token in model_name.split():
        if not token.endswith('GHz'):
            continue
        try:
            return float(token[:-3]) * 1000
        except ValueError:
            return None
    return None
