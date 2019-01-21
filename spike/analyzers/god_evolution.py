import re

def counter(metric_list):
    count = 0

    for i in range(1, len(metric_list)):
        if metric_list[i] > metric_list[i-1]:
            count += 1
        if metric_list[i] < metric_list[i-1]:
            count -= 1
    
    return count

with open('out.txt') as f:

    for line in f:
        last_wmc = 0
        if line.startswith("["):
            struct = re.search('](.*):', line).group(1).lstrip()

            wmc = []
            atfd = []
            tcc = []

            for d in f:
                if d.startswith('files'):
                    dsplit = d.split()
                    wmc.append(int(dsplit[2]))
                    atfd.append(int(dsplit[4]))
                    tcc.append(float(dsplit[6]))
                else:
                    break

            if len(wmc) <= 1: continue
            
            wmc_count = 0
            atfd_count = 0
            tcc_count = 0

            wmc_count = counter(wmc)
            atfd_count = counter(atfd)
            tcc_count = counter(tcc)

            print(struct, wmc_count, atfd_count, tcc_count)
