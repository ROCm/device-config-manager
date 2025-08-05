from "registry.test.pensando.io:5000/device-config-manager-build:v1.0"

user = getenv("USER")
group = getenv("GROUP_NAME")
uid = getenv("USER_UID")
gid = getenv("USER_GID")

# remove old version of go
run "rm -rf /usr/local/go"

# download go1.23
run "wget https://go.dev/dl/go1.23.4.linux-amd64.tar.gz && tar -C /usr/local/ -xzf go1.23.4.linux-amd64.tar.gz && rm go1.23.4.linux-amd64.tar.gz"
# download and install kubectl 
run "curl -LO https://dl.k8s.io/release/v1.30.0/bin/linux/amd64/kubectl"
run "chmod +x kubectl"
run "sudo mv kubectl /usr/local/bin"

run "curl -o /usr/bin/asset-pull http://pm.test.pensando.io/tools/asset-pull"
run "chmod +x /usr/bin/asset-pull"
run "curl -o /usr/bin/asset-push http://pm.test.pensando.io/tools/asset-push"
run "chmod +x /usr/bin/asset-push"
copy "asset-build/configmanager-asset-push.sh", "/configmanager-asset-push.sh"
run "chmod +x /configmanager-asset-push.sh"

if user == "root"
  # update user .bash_profile
  run "echo 'export GOPATH=/usr' >> /root/.bash_profile"
  run "echo 'export GOFLAGS=-mod=vendor' >> /root/.bash_profile"
  run "echo 'export PATH=/usr/local/go/bin:$PATH' >> /root/.bash_profile"

  run "localedef -i en_US -f UTF-8 en_US.UTF-8"
else
if user != ""
  # add user
  run "groupadd -g #{gid} #{group}"
  run "useradd -l -m -s /usr/bin/bash -u #{uid} -g #{gid} -G docker #{user}"

  # go installs in /usr, make it world writeable
  run "chmod 777 /usr/bin"

  # update user .bash_profile
  run "mkdir -p /home/#{user}"
  run "echo 'export GOPATH=/usr' >> /home/#{user}/.bash_profile"
  run "echo 'export PATH=/usr/local/go/bin:$PATH:/usr/local/bin' >> /home/#{user}/.bash_profile"
  run "echo 'export GOFLAGS=-mod=vendor' >> /home/#{user}/.bash_profile"
  run "echo 'sudo chown -R #{user} /device-config-manager/' >> /home/#{user}/.bash_profile"
  run "echo 'sudo chgrp -R #{user} /device-config-manager/' >> /home/#{user}/.bash_profile"
  run "echo 'Defaults secure_path = /usr/local/go/bin:$PATH:/bin:/usr/sbin/' >> /etc/sudoers"

  run "echo '#{user} ALL=(root) NOPASSWD:ALL' > /etc/sudoers.d/#{user} && chmod 0440 /etc/sudoers.d/#{user}"

  run "localedef -i en_US -f UTF-8 en_US.UTF-8"
end
end

env GOPATH: "/usr"
env GOFLAGS: "-mod=vendor"
run "git config --global --add safe.directory ${GOPATH}/src/github.com/ROCm/device-config-manager"

# A scratch pad file for exporting some host/workspace particulars into container, to be used for
# recording them into build packaging.
run "echo 'HOST_HOSTNAME=#{getenv("HOST_HOSTNAME")}' >> /usr/build_host_meta_data"
run "echo 'HOST_WORKSPACE=#{getenv("HOST_WORKSPACE")}' >> /usr/build_host_meta_data"

inside "/etc" do
  run "rm -f localtime"
  run "ln -s /usr/share/zoneinfo/US/Pacific localtime"
end

workdir "/device-config-manager"

copy "entrypoint.sh", "/entrypoint.sh"
run "chmod +x /entrypoint.sh"

entrypoint "/entrypoint.sh"
