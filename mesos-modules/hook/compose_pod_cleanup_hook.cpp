/**
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#include <mesos/hook.hpp>
#include <mesos/mesos.hpp>
#include <mesos/module.hpp>

#include <mesos/module/hook.hpp>

#include <process/future.hpp>
#include <process/process.hpp>
#include <process/protobuf.hpp>

#include <stout/foreach.hpp>
#include <stout/os.hpp>
#include <stout/try.hpp>
#include <sstream>

using namespace mesos;

using process::Future;


class ComposePodCleanupHook : public Hook
{
public:
  // This hook is called when the executor is being removed.
  virtual Try<Nothing> slaveRemoveExecutorHook(
      const FrameworkInfo& frameworkInfo,
      const ExecutorInfo& executorInfo)
  {
    LOG(INFO) << "Executing 'slaveRemoveExecutorHook'";
    
    std::string executorId = executorInfo.executor_id().value();
    std::string containerIdCommand = "docker ps -a --filter=\"label=executorId="+executorId+"\" -q 2>&1";
    std::list<std::string> result = exec(containerIdCommand);
    
    if(result.empty()) {
      LOG(INFO) << "ComposePodCleanupHook: nothing to clean";
      return Nothing();
    }
    
    std::string containers = join(result);
    stopContainer(containers);
    
    std::list<std::string>::iterator it;
    std::string parentId;
    for(it = result.begin();it != result.end();++it) {
        std::string cmd = "docker inspect --format {{.HostConfig.CgroupParent}} " + *it;
        std::list<std::string> cgParent = exec(cmd);
        if (!cgParent.empty()) {
          parentId = join(cgParent);
          break;
        }
    }

    // keep the containers for debugging purposes.
    //removeContainer(containers);

    if(!parentId.empty()){
      deleteCgroup(parentId);
    }

    return Nothing();
  }

  std::string join(std::list<std::string> l, const char* delim=" ") {
    std::ostringstream imploded;
    std::copy(l.begin(), l.end(),
           std::ostream_iterator<std::string>(imploded, delim));
    return imploded.str();
  }

  std::list<std::string> stopContainer(std::string containerId) {
    std::string command = "docker stop "+containerId;
    return exec(command);
  }

  std::list<std::string> removeContainer(std::string containerId) {
     std::string command = "docker rm -v "+containerId;
     return exec(command);
  }

  void deleteCgroup(std::string parentId) {
    std::string command = "lscgroup | grep " + parentId;
    std::list<std::string> groupList = exec(command);
    if(!groupList.empty()) {
      command = "cgdelete "+ join(groupList);
      exec(command);
    }
  }
  
  std::list<std::string> exec(std::string cmd) {
    char buffer[128];
    std::string line;
    std::list<std::string> result;
    FILE* pipe = popen(cmd.c_str(), "r");
    if (!pipe) {
      LOG(INFO)<<"popen() failed! cmd:" << cmd;
    }
    try {
      while (!feof(pipe)) {
        if (fgets(buffer, 128, pipe) != NULL){
          line.append(buffer);
          if(line.back() == '\n') {
            line.pop_back();
            result.push_back(std::move(line));
            line.clear();
          }
        }
      }
    } catch (...) {
        LOG(INFO) <<"caught execption while processing shell cmd" << cmd;
        pclose(pipe);
        result.clear();
    }
    
    if (pclose(pipe) < 0) {
       LOG(INFO) << "command failed with " << join(result);
       result.clear();
    }
     return result;
  }

};


static Hook* createHook(const Parameters& parameters)
{
  return new ComposePodCleanupHook();
}


// Declares a Hook module named 'org_apache_mesos_ComposePodCleanupHook'.
mesos::modules::Module<Hook> org_apache_mesos_ComposePodCleanupHook(
    MESOS_MODULE_API_VERSION,
    MESOS_VERSION,
    "Apache Mesos",
    "modules@mesos.apache.org",
    "Compose Pod Cleanup Hook module.",
    NULL,
    createHook);

