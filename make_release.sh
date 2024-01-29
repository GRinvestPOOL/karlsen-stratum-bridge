CMD_PATH="../cmd/nexelliabridge"
rm -rf release
mkdir -p release
cd release
VERSION=1.1.0
ARCHIVE="nxl_bridge-${VERSION}"
OUTFILE="nxl_bridge"
OUTDIR="nxl_bridge"

# windows
mkdir -p ${OUTDIR};env GOOS=windows GOARCH=amd64 go build -o ${OUTDIR}/${OUTFILE}.exe ${CMD_PATH};cp ${CMD_PATH}/config.yaml ${OUTDIR}/
zip -r ${ARCHIVE}.zip ${OUTDIR}
rm -rf ${OUTDIR}

# linux
mkdir -p ${OUTDIR};env GOOS=linux GOARCH=amd64 go build -o ${OUTDIR}/${OUTFILE} ${CMD_PATH};cp ${CMD_PATH}/config.yaml ${OUTDIR}/
tar -czvf ${ARCHIVE}.tar.gz ${OUTDIR}

# hive
cp ../misc/hive/* ${OUTDIR}
tar -czvf ${ARCHIVE}_hive.tar.gz ${OUTDIR}

# checksums
sha256sum ${ARCHIVE}.tar.gz ${ARCHIVE}.zip ${ARCHIVE}_hive.tar.gz > SHA256SUMS
