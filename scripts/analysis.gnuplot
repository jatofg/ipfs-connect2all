reset
set key top left
set xlabel "Time since start (d)"
set ylabel "# Peers"
# total numbers from stats files
plot 'total.dat' u (($0)/144):1 t "known" w l, "" u (($0)/144):2 t "connected" w l, "" u (($0)/144):3 t "established" w l, "" u (($0)/144):4 t "failed" w l, "" u (($0)/144):5 t "successful" w l
# total crawl numbers
plot 'comparison_0m.dat' u (($0)/4):1 t "crawl" w l, "" u (($0)/4):2 t "crawl-reachable" w l #, "" u (($0)/4):3 t "known" w l, "" u (($0)/4):4 t "connected" w l, "" u (($0)/4):5 t "successful" w l, "" u (($0)/4):6 t "failed" w l
# comparison of crawl and snapshot numbers - 0m, 10m, 20m, 30m after start of crawl
plot 'comparison_0m.dat' u (($0)/4):7 t "crawl-reachable, !known" w l, "" u (($0)/4):8 t "crawl-reachable, !connected" w l, "" u (($0)/4):9 t "crawl-reachable, !successful" w l, "" u (($0)/4):10 t "crawl-reachable, failed" w l
plot 'comparison_0m.dat' u (($0)/4):11 t "known, !crawl" w l, "" u (($0)/4):12 t "connected, !crawl" w l, "" u (($0)/4):13 t "connected, !crawl-reachable" w l, "" u (($0)/4):14 t "successful, !crawl" w l, "" u (($0)/4):15 t "successful, !crawl-reachable" w l
plot 'comparison_10m.dat' u (($0)/4):7 t "crawl-reachable, !known" w l, "" u (($0)/4):8 t "crawl-reachable, !connected" w l, "" u (($0)/4):9 t "crawl-reachable, !successful" w l, "" u (($0)/4):10 t "crawl-reachable, failed" w l
plot 'comparison_10m.dat' u (($0)/4):11 t "known, !crawl" w l, "" u (($0)/4):12 t "connected, !crawl" w l, "" u (($0)/4):13 t "connected, !crawl-reachable" w l, "" u (($0)/4):14 t "successful, !crawl" w l, "" u (($0)/4):15 t "successful, !crawl-reachable" w l
plot 'comparison_20m.dat' u (($0)/4):7 t "crawl-reachable, !known" w l, "" u (($0)/4):8 t "crawl-reachable, !connected" w l, "" u (($0)/4):9 t "crawl-reachable, !successful" w l, "" u (($0)/4):10 t "crawl-reachable, failed" w l
plot 'comparison_20m.dat' u (($0)/4):11 t "known, !crawl" w l, "" u (($0)/4):12 t "connected, !crawl" w l, "" u (($0)/4):13 t "connected, !crawl-reachable" w l, "" u (($0)/4):14 t "successful, !crawl" w l, "" u (($0)/4):15 t "successful, !crawl-reachable" w l
plot 'comparison_30m.dat' u (($0)/4):7 t "crawl-reachable, !known" w l, "" u (($0)/4):8 t "crawl-reachable, !connected" w l, "" u (($0)/4):9 t "crawl-reachable, !successful" w l, "" u (($0)/4):10 t "crawl-reachable, failed" w l
plot 'comparison_30m.dat' u (($0)/4):11 t "known, !crawl" w l, "" u (($0)/4):12 t "connected, !crawl" w l, "" u (($0)/4):13 t "connected, !crawl-reachable" w l, "" u (($0)/4):14 t "successful, !crawl" w l, "" u (($0)/4):15 t "successful, !crawl-reachable" w l
# churn calculations
set key top right
#set yrange [0:3000]
set ylabel "# Peers per 10 min"
plot 'churn.dat' u (($0)/144):1 t "newly known" w l, '' u (($0)/144):2 t "new connections" w l, '' u (($0)/144):3 t "lost connections" w l
